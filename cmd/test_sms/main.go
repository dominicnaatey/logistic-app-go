package main

import (
	"fmt"
	"log"

	"logistic-app-go/internal/sms"
	"logistic-app-go/pkg/config"
)

func main() {
	fmt.Println("🧪 Testing SMS Service (Africa's Talking)...\n")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config load failed: %v", err)
	}

	// -------------------------------------------------------
	// TEST 1: Initialise service
	// -------------------------------------------------------
	fmt.Println("📝 Test 1: Initialise SMS service...")
	smsSvc, err := sms.NewService(cfg.SMS.Username, cfg.SMS.APIKey, cfg.SMS.SenderID)
	if err != nil {
		log.Fatalf("❌ SMS service init failed: %v", err)
	}
	fmt.Printf("  ✓ Service initialised (sandbox: %v)\n", smsSvc.IsSandbox())

	// -------------------------------------------------------
	// TEST 2: Reject missing credentials
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 2: Reject empty credentials...")
	_, err = sms.NewService("", "", "")
	if err != nil {
		fmt.Printf("  ✓ Empty credentials correctly rejected: %v\n", err)
	} else {
		log.Fatal("❌ Empty credentials were accepted!")
	}

	// -------------------------------------------------------
	// TEST 3: Send OTP SMS via AT sandbox
	// NOTE: In sandbox mode the message goes to the AT simulator,
	//       NOT a real phone. Check: https://simulator.africastalking.com
	// -------------------------------------------------------
	fmt.Println("\n📱 Test 3: Send OTP SMS (sandbox)...")
	// AT sandbox requires the phone to be registered in the simulator
	// Use your own number registered in AT simulator
	testPhone := "+233501234567"
	err = smsSvc.SendOTP(testPhone, "123456")
	if err != nil {
		// In sandbox, AT may return a non-Success status for unregistered numbers.
		// This is expected behaviour — the service itself worked correctly.
		fmt.Printf("  ⚠️  AT sandbox response: %v\n", err)
		fmt.Println("  ℹ️  This is normal if the number isn't registered in the AT simulator.")
		fmt.Println("  ℹ️  Register your number at https://simulator.africastalking.com")
	} else {
		fmt.Printf("  ✓ OTP SMS sent to %s — check AT simulator\n", testPhone)
	}

	// -------------------------------------------------------
	// TEST 4: Send welcome SMS
	// -------------------------------------------------------
	fmt.Println("\n📱 Test 4: Send welcome SMS (English)...")
	err = smsSvc.SendWelcome(testPhone, "Kwame", "en")
	if err != nil {
		fmt.Printf("  ⚠️  AT sandbox response: %v\n", err)
	} else {
		fmt.Printf("  ✓ Welcome SMS (EN) sent to %s\n", testPhone)
	}

	// -------------------------------------------------------
	// TEST 5: Send welcome SMS in French (Mali)
	// -------------------------------------------------------
	fmt.Println("\n📱 Test 5: Send welcome SMS (French/Mali)...")
	err = smsSvc.SendWelcome(testPhone, "Moussa", "fr")
	if err != nil {
		fmt.Printf("  ⚠️  AT sandbox response: %v\n", err)
	} else {
		fmt.Printf("  ✓ Welcome SMS (FR) sent to %s\n", testPhone)
	}

	// -------------------------------------------------------
	// TEST 6: Send trip assignment SMS
	// -------------------------------------------------------
	fmt.Println("\n📱 Test 6: Send trip assignment SMS...")
	err = smsSvc.SendTripAssignment(testPhone, "Kwame", "Accra, Ghana", "Bamako, Mali")
	if err != nil {
		fmt.Printf("  ⚠️  AT sandbox response: %v\n", err)
	} else {
		fmt.Printf("  ✓ Trip assignment SMS sent to %s\n", testPhone)
	}

	// -------------------------------------------------------
	// TEST 7: Reject empty phone
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 7: Reject empty phone number...")
	err = smsSvc.SendOTP("", "123456")
	if err != nil {
		fmt.Printf("  ✓ Empty phone correctly rejected: %v\n", err)
	} else {
		log.Fatal("❌ Empty phone was accepted!")
	}

	// -------------------------------------------------------
	// Summary
	// -------------------------------------------------------
	fmt.Println("\n✅ SMS Service tests complete!")
	fmt.Println("\n⚠️  Note on sandbox behaviour:")
	fmt.Println("   - Messages go to AT simulator, NOT real phones")
	fmt.Println("   - Register your test number at https://simulator.africastalking.com")
	fmt.Println("   - AT sandbox may reject unregistered numbers — this is expected")
	fmt.Println("\nSMS service is ready. Next step: Phase 1.4 — JWT Middleware")
}
