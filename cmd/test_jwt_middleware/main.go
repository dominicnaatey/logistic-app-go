package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authMiddleware "logistic-app-go/internal/auth"
	"logistic-app-go/internal/user"
	"logistic-app-go/pkg/auth"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/response"
)

func main() {
	fmt.Println("🧪 Testing JWT Middleware...")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config load failed: %v", err)
	}

	jwtManager, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
	if err != nil {
		log.Fatalf("❌ JWT manager init failed: %v", err)
	}

	// Set Gin to test mode (suppresses debug logs)
	gin.SetMode(gin.TestMode)

	// Build a test router that mirrors the real server setup
	router := buildTestRouter(jwtManager)

	// -------------------------------------------------------
	// TEST 1: No Authorization header → 401
	// -------------------------------------------------------
	fmt.Println("🚫 Test 1: No Authorization header → 401...")
	resp := doRequest(router, "GET", "/protected", "")
	assertStatus(resp, 401, "no auth header")

	// -------------------------------------------------------
	// TEST 2: Wrong format → 401
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 2: Wrong header format → 401...")
	resp = doRequest(router, "GET", "/protected", "Token abc123")
	assertStatus(resp, 401, "wrong format")

	// -------------------------------------------------------
	// TEST 3: Tampered token → 401
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 3: Tampered token → 401...")
	resp = doRequest(router, "GET", "/protected", "Bearer eyJhbGciOiJIUzI1NiJ9.tampered.signature")
	assertStatus(resp, 401, "tampered token")

	// -------------------------------------------------------
	// TEST 4: Valid token → 200 + claims accessible
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 4: Valid token → 200 + claims in response...")
	testID := uuid.New()
	testPhone := "+233501234567"
	testRole := user.RoleDriver

	token, err := jwtManager.Generate(testID, testPhone, testRole)
	if err != nil {
		log.Fatalf("❌ Token generation failed: %v", err)
	}

	resp = doRequest(router, "GET", "/protected", "Bearer "+token)
	assertStatus(resp, 200, "valid token")

	body := parseBody(resp)
	data := body["data"].(map[string]interface{})
	if data["user_id"] != testID.String() {
		log.Fatalf("❌ user_id mismatch: got %v", data["user_id"])
	}
	if data["phone"] != testPhone {
		log.Fatalf("❌ phone mismatch: got %v", data["phone"])
	}
	if data["role"] != testRole {
		log.Fatalf("❌ role mismatch: got %v", data["role"])
	}
	fmt.Printf("  ✓ user_id: %v\n", data["user_id"])
	fmt.Printf("  ✓ phone:   %v\n", data["phone"])
	fmt.Printf("  ✓ role:    %v\n", data["role"])

	// -------------------------------------------------------
	// TEST 5: RBAC — correct role allowed
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 5: RBAC — driver accessing driver route → 200...")
	resp = doRequest(router, "GET", "/driver-only", "Bearer "+token)
	assertStatus(resp, 200, "driver role allowed")

	// -------------------------------------------------------
	// TEST 6: RBAC — wrong role blocked → 403
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 6: RBAC — driver accessing admin route → 403...")
	resp = doRequest(router, "GET", "/admin-only", "Bearer "+token)
	assertStatus(resp, 403, "driver blocked from admin route")

	// -------------------------------------------------------
	// TEST 7: RBAC — admin token accessing admin route → 200
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 7: RBAC — admin token accessing admin route → 200...")
	adminToken, _ := jwtManager.Generate(uuid.New(), "+233509999999", user.RoleAdmin)
	resp = doRequest(router, "GET", "/admin-only", "Bearer "+adminToken)
	assertStatus(resp, 200, "admin role allowed")

	// -------------------------------------------------------
	// TEST 8: RBAC — multi-role route (driver OR owner_operator)
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 8: RBAC — owner_operator accessing multi-role route → 200...")
	opToken, _ := jwtManager.Generate(uuid.New(), "+22370000001", user.RoleOwnerOperator)
	resp = doRequest(router, "GET", "/drivers-and-operators", "Bearer "+opToken)
	assertStatus(resp, 200, "owner_operator allowed on multi-role route")

	fmt.Println("\n🚫 Test 8b: RBAC — shipper blocked from multi-role route → 403...")
	shipperToken, _ := jwtManager.Generate(uuid.New(), "+233501112222", user.RoleShipper)
	resp = doRequest(router, "GET", "/drivers-and-operators", "Bearer "+shipperToken)
	assertStatus(resp, 403, "shipper blocked from driver/operator route")

	// -------------------------------------------------------
	// TEST 9: Public route requires no token
	// -------------------------------------------------------
	fmt.Println("\n✅ Test 9: Public route — no token needed → 200...")
	resp = doRequest(router, "GET", "/public", "")
	assertStatus(resp, 200, "public route accessible")

	// -------------------------------------------------------
	// Summary
	// -------------------------------------------------------
	fmt.Println("\n✅ All JWT Middleware tests passed!")
	fmt.Println("\nJWT middleware is ready. Next step: Phase 1.5 — Auth Handlers (send-otp + verify-otp)")
}

// buildTestRouter mirrors how the real server sets up protected/public routes.
func buildTestRouter(jwtManager *auth.JWTManager) *gin.Engine {
	router := gin.New()

	// Public route — no auth needed
	router.GET("/public", func(c *gin.Context) {
		response.Success(c, gin.H{"message": "public endpoint"})
	})

	// Protected group — JWT required
	protected := router.Group("/")
	protected.Use(authMiddleware.JWTMiddleware(jwtManager))
	{
		protected.GET("/protected", func(c *gin.Context) {
			claims := authMiddleware.GetClaims(c)
			response.Success(c, gin.H{
				"user_id": claims.UserID.String(),
				"phone":   claims.Phone,
				"role":    claims.Role,
			})
		})

		// Driver-only route
		driverRoutes := protected.Group("/")
		driverRoutes.Use(authMiddleware.RequireRoles(user.RoleDriver))
		driverRoutes.GET("/driver-only", func(c *gin.Context) {
			response.Success(c, gin.H{"message": "driver endpoint"})
		})

		// Admin-only route
		adminRoutes := protected.Group("/")
		adminRoutes.Use(authMiddleware.RequireRoles(user.RoleAdmin))
		adminRoutes.GET("/admin-only", func(c *gin.Context) {
			response.Success(c, gin.H{"message": "admin endpoint"})
		})

		// Drivers + owner-operators route
		driverOpRoutes := protected.Group("/")
		driverOpRoutes.Use(authMiddleware.RequireRoles(user.RoleDriver, user.RoleOwnerOperator))
		driverOpRoutes.GET("/drivers-and-operators", func(c *gin.Context) {
			response.Success(c, gin.H{"message": "driver/operator endpoint"})
		})
	}

	return router
}

// doRequest fires an HTTP request against the test router.
func doRequest(router *gin.Engine, method, path, authHeader string) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result()
}

// assertStatus checks the HTTP status and prints result.
func assertStatus(resp *http.Response, expected int, label string) {
	if resp.StatusCode != expected {
		log.Fatalf("❌ [%s] expected %d, got %d", label, expected, resp.StatusCode)
	}
	fmt.Printf("  ✓ Status %d as expected\n", resp.StatusCode)
}

// parseBody reads and JSON-decodes the response body.
func parseBody(resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	return out
}
