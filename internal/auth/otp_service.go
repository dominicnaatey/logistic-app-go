package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"logistic-app-go/pkg/cache"
)

const (
	// otpTTL is how long an OTP remains valid after being issued.
	otpTTL = 10 * time.Minute

	// otpRateTTL is the window in which rate limiting is enforced.
	otpRateTTL = 10 * time.Minute

	// otpMaxAttempts is the maximum number of OTP requests per phone per window.
	otpMaxAttempts = 3

	// otpLength is the number of digits in the OTP code.
	otpLength = 6
)

// OTPService handles OTP generation, storage and verification via a cache backend.
// Redis key scheme:
//
//	otp:{phone}       → the 6-digit code, expires after otpTTL
//	otp:rate:{phone}  → request counter, expires after otpRateTTL
type OTPService struct {
	cache cache.Cache
}

// NewOTPService creates a new OTPService backed by any cache.Cache implementation.
// In production this is Upstash Redis; in tests it can be an in-memory stub.
func NewOTPService(c cache.Cache) *OTPService {
	return &OTPService{cache: c}
}

// GenerateAndStore creates a new 6-digit OTP, stores it in Redis with a
// 10-minute TTL, and returns the code so it can be sent via SMS.
// It enforces a rate limit of 3 requests per phone per 10-minute window.
func (s *OTPService) GenerateAndStore(ctx context.Context, phone string) (string, error) {
	// Check rate limit before generating a new code
	if err := s.checkRateLimit(ctx, phone); err != nil {
		return "", err
	}

	// Generate cryptographically secure 6-digit code
	code, err := generateCode(otpLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Store the code in Redis: SET otp:{phone} {code} EX 600
	otpKey := otpKey(phone)
	if err := s.cache.Set(ctx, otpKey, code, otpTTL); err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	// Increment the rate-limit counter
	rateKey := rateKey(phone)
	count, err := s.cache.Incr(ctx, rateKey)
	if err != nil {
		return "", fmt.Errorf("failed to update rate counter: %w", err)
	}
	// Set TTL on first increment only
	if count == 1 {
		if err := s.cache.Expire(ctx, rateKey, otpRateTTL); err != nil {
			return "", fmt.Errorf("failed to set rate TTL: %w", err)
		}
	}

	return code, nil
}

// Verify checks the supplied code against what is stored in Redis.
// On a successful match the code is immediately deleted (single-use).
// Returns nil on success, a descriptive error otherwise.
func (s *OTPService) Verify(ctx context.Context, phone, code string) error {
	otpKey := otpKey(phone)

	stored, err := s.cache.Get(ctx, otpKey)
	if err != nil {
		// Redis returns "key not found" when TTL has expired or key never existed
		return fmt.Errorf("OTP expired or not found — please request a new one")
	}

	if stored != code {
		return fmt.Errorf("invalid OTP code")
	}

	// Delete immediately — OTPs are single-use
	if err := s.cache.Del(ctx, otpKey); err != nil {
		// Non-fatal: code matched, proceed even if delete fails
		_ = err
	}

	return nil
}

// RemainingAttempts returns how many more OTP requests the phone number
// can make in the current rate-limit window.
func (s *OTPService) RemainingAttempts(ctx context.Context, phone string) (int, error) {
	rateKey := rateKey(phone)
	val, err := s.cache.Get(ctx, rateKey)
	if err != nil {
		// Key does not exist yet → full quota available
		return otpMaxAttempts, nil
	}

	var count int
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, fmt.Errorf("unexpected rate counter value: %s", val)
	}

	remaining := otpMaxAttempts - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// checkRateLimit returns an error if the phone has exceeded the allowed
// number of OTP requests within the current time window.
func (s *OTPService) checkRateLimit(ctx context.Context, phone string) error {
	rateKey := rateKey(phone)
	val, err := s.cache.Get(ctx, rateKey)
	if err != nil {
		// Key not present → no requests yet, allow
		return nil
	}

	var count int
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return fmt.Errorf("unexpected rate counter value: %s", val)
	}

	if count >= otpMaxAttempts {
		return fmt.Errorf(
			"too many OTP requests — maximum %d per %v, please try again later",
			otpMaxAttempts,
			otpRateTTL,
		)
	}
	return nil
}

// generateCode returns a zero-padded n-digit random numeric string using
// crypto/rand so it is suitable for security-sensitive OTPs.
func generateCode(digits int) (string, error) {
	// Calculate upper bound: 10^digits (e.g. 1_000_000 for 6 digits)
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Zero-pad to ensure consistent length (e.g. "007341" not "7341")
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

// otpKey returns the Redis key for storing an OTP code.
func otpKey(phone string) string {
	return fmt.Sprintf("otp:%s", phone)
}

// rateKey returns the Redis key for the rate-limit counter.
func rateKey(phone string) string {
	return fmt.Sprintf("otp:rate:%s", phone)
}
