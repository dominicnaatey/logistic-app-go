package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	internalAuth "logistic-app-go/pkg/auth"
	"logistic-app-go/pkg/response"
	"logistic-app-go/internal/sms"
	"logistic-app-go/internal/user"
)

// Handler holds all dependencies needed by the auth HTTP handlers.
type Handler struct {
	otpSvc     *OTPService
	smsSvc     *sms.Service
	userSvc    user.Service
	jwtManager *internalAuth.JWTManager
}

// NewHandler creates a new auth Handler with all required dependencies.
func NewHandler(
	otpSvc *OTPService,
	smsSvc *sms.Service,
	userSvc user.Service,
	jwtManager *internalAuth.JWTManager,
) *Handler {
	return &Handler{
		otpSvc:     otpSvc,
		smsSvc:     smsSvc,
		userSvc:    userSvc,
		jwtManager: jwtManager,
	}
}

// RegisterRoutes registers all auth endpoints on the given router group.
// Public routes (no JWT required):
//
//	POST /auth/send-otp
//	POST /auth/verify-otp
//
// Protected routes (JWT required — pass the authed group):
//
//	GET  /auth/me
func (h *Handler) RegisterRoutes(public *gin.RouterGroup, protected *gin.RouterGroup) {
	public.POST("/auth/send-otp", h.SendOTP)
	public.POST("/auth/verify-otp", h.VerifyOTP)
	protected.GET("/auth/me", h.Me)
}

// ──────────────────────────────────────────────────────────────
// Request / Response DTOs
// ──────────────────────────────────────────────────────────────

// SendOTPRequest is the request body for POST /auth/send-otp.
type SendOTPRequest struct {
	// Phone must be in E.164 format, e.g. +233501234567
	Phone string `json:"phone" binding:"required"`

	// Role must be one of the defined constants in the user package.
	// Sent at first login so we know what kind of account to create.
	Role string `json:"role" binding:"required"`

	// Language preference: "en" (default) or "fr" (Mali)
	Language string `json:"language"`
}

// VerifyOTPRequest is the request body for POST /auth/verify-otp.
type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code"  binding:"required"`
}

// ──────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────

// SendOTP godoc
// @Summary  Request an OTP via SMS
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body SendOTPRequest true "Phone + role"
// @Success  200
// @Router   /auth/send-otp [post]
//
// SendOTP validates the request, enforces rate limiting, generates a
// 6-digit code, stores it in Redis, and delivers it via Africa's Talking SMS.
func (h *Handler) SendOTP(c *gin.Context) {
	var req SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "phone and role are required")
		return
	}

	// Basic E.164 check (must start with +)
	if len(req.Phone) < 8 || req.Phone[0] != '+' {
		response.BadRequest(c, "phone must be in E.164 format (e.g. +233501234567)")
		return
	}

	// Generate + store OTP (also enforces rate limit)
	code, err := h.otpSvc.GenerateAndStore(c.Request.Context(), req.Phone)
	if err != nil {
		// Rate limit errors are user-facing; surface them as 429
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Send via SMS
	if err := h.smsSvc.SendOTP(req.Phone, code); err != nil {
		// SMS failure should not expose internal details to client
		response.Error(c, http.StatusServiceUnavailable, "failed to send SMS, please try again")
		return
	}

	response.Success(c, gin.H{
		"message": "OTP sent successfully",
		"phone":   req.Phone,
	})
}

// VerifyOTP godoc
// @Summary  Verify OTP and receive JWT
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body VerifyOTPRequest true "Phone + OTP code"
// @Success  200
// @Router   /auth/verify-otp [post]
//
// VerifyOTP checks the submitted code against Redis, creates or fetches
// the user record, and returns a signed JWT.
func (h *Handler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "phone and code are required")
		return
	}

	// Verify OTP against Redis (single-use, auto-deleted on match)
	if err := h.otpSvc.Verify(c.Request.Context(), req.Phone, req.Code); err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// We need the role to create the user if this is their first login.
	// Read it from query param (set during send-otp step) or default to shipper.
	role := c.Query("role")
	if role == "" {
		role = user.RoleShipper
	}
	language := c.Query("language")

	// Find existing user or create new one
	u, isNew, err := h.userSvc.FindOrCreate(req.Phone, role, language)
	if err != nil {
		response.InternalError(c, "failed to retrieve user account")
		return
	}

	// Issue JWT
	token, err := h.jwtManager.Generate(u.ID, u.Phone, u.Role)
	if err != nil {
		response.InternalError(c, "failed to issue token")
		return
	}

	// Send welcome SMS on first login (non-blocking — ignore errors)
	if isNew {
		go func() {
			_ = h.smsSvc.SendWelcome(u.Phone, u.Name, u.Language)
		}()
	}

	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}

	c.JSON(status, gin.H{
		"success":  true,
		"token":    token,
		"is_new":   isNew,
		"user": gin.H{
			"id":         u.ID,
			"phone":      u.Phone,
			"role":       u.Role,
			"language":   u.Language,
			"kyc_status": u.KYCStatus,
		},
	})
}

// Me godoc
// @Summary  Get current authenticated user
// @Tags     auth
// @Security BearerAuth
// @Produce  json
// @Success  200
// @Router   /auth/me [get]
//
// Me returns the full profile of the authenticated user.
// Requires a valid JWT in the Authorization header.
func (h *Handler) Me(c *gin.Context) {
	claims := GetClaims(c)
	if claims == nil {
		response.Unauthorized(c, "not authenticated")
		return
	}

	u, err := h.userSvc.GetByID(claims.UserID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, u)
}
