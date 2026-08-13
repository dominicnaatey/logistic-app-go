package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	internalAuth "logistic-app-go/internal/auth"
	"logistic-app-go/internal/sms"
	"logistic-app-go/internal/user"
	"logistic-app-go/pkg/auth"
	"logistic-app-go/pkg/cache"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/database"
	"logistic-app-go/pkg/response"
)

func main() {
	fmt.Println("🧪 Testing Auth Handlers (send-otp → verify-otp → /me)...")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config: %v", err)
	}

	db, err := database.NewPostgresDB(cfg.Database.URL, false)
	if err != nil {
		log.Fatalf("❌ DB: %v", err)
	}
	defer database.Close(db)
	database.Migrate(db)

	redisClient, err := cache.NewUpstashClient(cfg.Redis.UpstashRestURL, cfg.Redis.UpstashRestToken)
	if err != nil {
		log.Fatalf("❌ Redis: %v", err)
	}

	jwtManager, _ := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
	smsSvc, _ := sms.NewService(cfg.SMS.Username, cfg.SMS.APIKey, cfg.SMS.SenderID)
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	otpSvc := internalAuth.NewOTPService(redisClient)
	authHandler := internalAuth.NewHandler(otpSvc, smsSvc, userSvc, jwtManager)

	gin.SetMode(gin.TestMode)
	router := buildRouter(jwtManager, authHandler)

	testPhone := "+233501888001"

	// Clean up any leftover state from previous runs
	redisClient.Del(context.Background(), "otp:"+testPhone, "otp:rate:"+testPhone)
	db.Unscoped().Where("phone = ?", testPhone).Delete(&user.User{})

	// -------------------------------------------------------
	// TEST 1: send-otp with missing body → 400
	// -------------------------------------------------------
	fmt.Println("🚫 Test 1: send-otp — missing body → 400...")
	resp := post(router, "/api/v1/auth/send-otp", nil, "")
	assertStatus(resp, 400, "missing body")

	// -------------------------------------------------------
	// TEST 2: send-otp with invalid phone → 400
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 2: send-otp — invalid phone → 400...")
	resp = post(router, "/api/v1/auth/send-otp", map[string]string{
		"phone": "0501234567", // missing leading +
		"role":  "driver",
	}, "")
	assertStatus(resp, 400, "invalid phone")

	// -------------------------------------------------------
	// TEST 3: send-otp success → 200
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 3: send-otp — valid request → 200...")
	resp = post(router, "/api/v1/auth/send-otp", map[string]string{
		"phone":    testPhone,
		"role":     user.RoleDriver,
		"language": user.LangEnglish,
	}, "")
	assertStatus(resp, 200, "send-otp success")
	fmt.Println("  ✓ OTP sent (check AT simulator)")

	// -------------------------------------------------------
	// TEST 4: verify-otp with wrong code → 401
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 4: verify-otp — wrong code → 401...")
	resp = post(router, "/api/v1/auth/verify-otp?role=driver&language=en", map[string]string{
		"phone": testPhone,
		"code":  "000000",
	}, "")
	assertStatus(resp, 401, "wrong OTP code")

	// -------------------------------------------------------
	// TEST 5: verify-otp with correct code → 200/201 + JWT
	// We read the real OTP directly from Redis for testing
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 5: verify-otp — correct code → JWT issued...")
	realCode, err := redisClient.Get(context.Background(), "otp:"+testPhone)
	if err != nil {
		log.Fatalf("❌ Could not read OTP from Redis: %v", err)
	}
	fmt.Printf("  (using real OTP from Redis: %s)\n", realCode)

	resp = post(router, "/api/v1/auth/verify-otp?role=driver&language=en", map[string]string{
		"phone": testPhone,
		"code":  realCode,
	}, "")
	// 201 for new user, 200 for returning user
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Fatalf("❌ Expected 200/201, got %d", resp.StatusCode)
	}
	fmt.Printf("  ✓ Status %d (201 = new user, 200 = returning)\n", resp.StatusCode)

	body := parseBody(resp)
	token, ok := body["token"].(string)
	if !ok || token == "" {
		log.Fatalf("❌ No token in response: %v", body)
	}
	fmt.Printf("  ✓ JWT token issued (%d chars)\n", len(token))

	isNew := body["is_new"].(bool)
	fmt.Printf("  ✓ is_new: %v\n", isNew)

	userData := body["user"].(map[string]interface{})
	fmt.Printf("  ✓ user.phone: %v\n", userData["phone"])
	fmt.Printf("  ✓ user.role:  %v\n", userData["role"])

	// -------------------------------------------------------
	// TEST 6: /me without token → 401
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 6: /me — no token → 401...")
	resp = get(router, "/api/v1/auth/me", "")
	assertStatus(resp, 401, "/me without token")

	// -------------------------------------------------------
	// TEST 7: /me with valid token → 200 + user data
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 7: /me — valid token → 200 + user profile...")
	resp = get(router, "/api/v1/auth/me", "Bearer "+token)
	assertStatus(resp, 200, "/me with token")

	meBody := parseBody(resp)
	meData := meBody["data"].(map[string]interface{})
	if meData["phone"] != testPhone {
		log.Fatalf("❌ /me returned wrong phone: %v", meData["phone"])
	}
	fmt.Printf("  ✓ phone:      %v\n", meData["phone"])
	fmt.Printf("  ✓ role:       %v\n", meData["role"])
	fmt.Printf("  ✓ kyc_status: %v\n", meData["kyc_status"])
	fmt.Printf("  ✓ language:   %v\n", meData["language"])

	// -------------------------------------------------------
	// TEST 8: second login returns existing user (is_new=false)
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 8: second login — returns existing user (is_new=false)...")

	// Request new OTP for same phone
	redisClient.Del(context.Background(), "otp:rate:"+testPhone)
	post(router, "/api/v1/auth/send-otp", map[string]string{
		"phone": testPhone, "role": user.RoleDriver,
	}, "")

	code2, _ := redisClient.Get(context.Background(), "otp:"+testPhone)
	resp = post(router, "/api/v1/auth/verify-otp?role=driver", map[string]string{
		"phone": testPhone,
		"code":  code2,
	}, "")
	assertStatus(resp, 200, "second login")
	body2 := parseBody(resp)
	if body2["is_new"].(bool) {
		log.Fatal("❌ Second login should not create a new user")
	}
	fmt.Println("  ✓ Existing user returned, is_new=false")

	// -------------------------------------------------------
	// Cleanup
	// -------------------------------------------------------
	fmt.Println("\n🧹 Cleanup...")
	redisClient.Del(context.Background(), "otp:"+testPhone, "otp:rate:"+testPhone)
	db.Unscoped().Where("phone = ?", testPhone).Delete(&user.User{})
	fmt.Println("  ✓ Test data removed")

	fmt.Println("\n✅ All Auth Handler tests passed!")
	fmt.Println("\nPhase 1 is complete. All steps verified:")
	fmt.Println("  ✅ 1.1 User Model")
	fmt.Println("  ✅ 1.2 OTP Service")
	fmt.Println("  ✅ 1.3 SMS Service")
	fmt.Println("  ✅ 1.4 JWT Middleware")
	fmt.Println("  ✅ 1.5 User Repository & Service")
	fmt.Println("  ✅ 1.6 Auth Handlers")
}

func buildRouter(jwtManager *auth.JWTManager, authHandler *internalAuth.Handler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "healthy"})
	})

	v1 := router.Group("/api/v1")
	protected := v1.Group("/", internalAuth.JWTMiddleware(jwtManager))
	authHandler.RegisterRoutes(v1, protected)

	return router
}

func post(router *gin.Engine, path string, body interface{}, token string) *http.Response {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result()
}

func get(router *gin.Engine, path, token string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result()
}

func assertStatus(resp *http.Response, expected int, label string) {
	if resp.StatusCode != expected {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("❌ [%s] expected %d, got %d — body: %s", label, expected, resp.StatusCode, b)
	}
	fmt.Printf("  ✓ Status %d as expected\n", resp.StatusCode)
}

func parseBody(resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	return out
}
