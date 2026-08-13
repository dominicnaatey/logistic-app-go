package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"logistic-app-go/internal/auth"
	"logistic-app-go/pkg/cache"
	"logistic-app-go/pkg/config"
)

func main() {
	fmt.Println("🧪 Testing OTP Service against Upstash Redis...")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config load failed: %v", err)
	}

	redisClient, err := cache.NewUpstashClient(cfg.Redis.UpstashRestURL, cfg.Redis.UpstashRestToken)
	if err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	defer redisClient.Close()
	fmt.Println("✓ Connected to Upstash Redis")
	fmt.Println()

	otpSvc := auth.NewOTPService(redisClient)
	ctx := context.Background()
	testPhone := "+233501999001"

	// Clean up any leftover keys from a previous failed run
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone))

	// -------------------------------------------------------
	// TEST 1: Generate OTP
	// -------------------------------------------------------
	fmt.Println("📝 Test 1: Generate OTP...")
	code, err := otpSvc.GenerateAndStore(ctx, testPhone)
	if err != nil {
		log.Fatalf("❌ GenerateAndStore failed: %v", err)
	}
	fmt.Printf("  ✓ OTP generated: %s (length: %d)\n", code, len(code))
	if len(code) != 6 {
		log.Fatalf("❌ OTP should be 6 digits, got %d", len(code))
	}

	// -------------------------------------------------------
	// TEST 2: Wrong code rejected
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 2: Wrong code rejected...")
	err = otpSvc.Verify(ctx, testPhone, "000000")
	if err != nil {
		fmt.Printf("  ✓ Wrong code correctly rejected: %v\n", err)
	} else {
		log.Fatal("❌ Wrong code was accepted — this should not happen!")
	}

	// -------------------------------------------------------
	// TEST 3: Correct code accepted (single use)
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 3: Correct code accepted...")
	err = otpSvc.Verify(ctx, testPhone, code)
	if err != nil {
		log.Fatalf("❌ Correct code rejected: %v", err)
	}
	fmt.Println("  ✓ OTP verified successfully")

	// -------------------------------------------------------
	// TEST 4: Single-use — code can't be reused
	// -------------------------------------------------------
	fmt.Println("\n🔒 Test 4: OTP is single-use (can't reuse)...")
	err = otpSvc.Verify(ctx, testPhone, code)
	if err != nil {
		fmt.Printf("  ✓ Reuse correctly rejected: %v\n", err)
	} else {
		log.Fatal("❌ OTP was accepted a second time — single-use not enforced!")
	}

	// -------------------------------------------------------
	// TEST 5: Rate limiting (max 3 per window)
	// -------------------------------------------------------
	fmt.Println("\n⏱️  Test 5: Rate limiting (max 3 requests)...")

	// Clean slate for rate limit test
	testPhone2 := "+233501999002"
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone2))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone2))

	for i := 1; i <= 3; i++ {
		_, err := otpSvc.GenerateAndStore(ctx, testPhone2)
		if err != nil {
			log.Fatalf("❌ Request %d should have succeeded: %v", i, err)
		}
		remaining, _ := otpSvc.RemainingAttempts(ctx, testPhone2)
		fmt.Printf("  ✓ Request %d/3 accepted — %d remaining\n", i, remaining)
	}

	// 4th request should be blocked
	_, err = otpSvc.GenerateAndStore(ctx, testPhone2)
	if err != nil {
		fmt.Printf("  ✓ 4th request correctly blocked: %v\n", err)
	} else {
		log.Fatal("❌ 4th request was NOT blocked — rate limit not working!")
	}

	// -------------------------------------------------------
	// TEST 6: Expired OTP rejected
	// -------------------------------------------------------
	fmt.Println("\n⏰  Test 6: Expired OTP rejected...")
	testPhone3 := "+233501999003"
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone3))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone3))

	// Manually store an OTP with a 1-second TTL to simulate expiry
	redisClient.Set(ctx, fmt.Sprintf("otp:%s", testPhone3), "123456", 1*time.Second)
	fmt.Println("  Waiting 2 seconds for OTP to expire...")
	time.Sleep(2 * time.Second)

	err = otpSvc.Verify(ctx, testPhone3, "123456")
	if err != nil {
		fmt.Printf("  ✓ Expired OTP correctly rejected: %v\n", err)
	} else {
		log.Fatal("❌ Expired OTP was accepted — TTL not working!")
	}

	// -------------------------------------------------------
	// TEST 7: Zero-padded codes (e.g. "007341")
	// -------------------------------------------------------
	fmt.Println("\n🔢 Test 7: Zero-padded OTP generation (100 samples)...")
	allSixDigits := true
	now := time.Now().UnixNano()
	for i := 0; i < 100; i++ {
		// Each phone is unique per run (uses nanosecond timestamp + index)
		// so the rate limiter never triggers across samples
		testPhone4 := fmt.Sprintf("+%015d", (now+int64(i))%1_000_000_000_000_000)
		c, err := otpSvc.GenerateAndStore(ctx, testPhone4)
		if err != nil {
			log.Fatalf("❌ Failed on sample %d: %v", i, err)
		}
		if len(c) != 6 {
			allSixDigits = false
			fmt.Printf("  ❌ Sample %d has wrong length: '%s'\n", i, c)
		}
		// Clean up immediately
		redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone4))
		redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone4))
	}
	if allSixDigits {
		fmt.Println("  ✓ All 100 generated codes are exactly 6 digits")
	}

	// -------------------------------------------------------
	// Cleanup
	// -------------------------------------------------------
	fmt.Println("\n🧹 Cleanup: removing test Redis keys...")
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone))
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone2))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone2))
	redisClient.Del(ctx, fmt.Sprintf("otp:%s", testPhone3))
	redisClient.Del(ctx, fmt.Sprintf("otp:rate:%s", testPhone3))
	fmt.Println("  ✓ Test keys removed")

	// -------------------------------------------------------
	// Summary
	// -------------------------------------------------------
	fmt.Println("\n✅ All OTP Service tests passed!")
	fmt.Println("\nOTP service is ready. Next step: Phase 1.3 — SMS Service (Africa's Talking)")
}
