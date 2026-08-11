package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	SMS      SMSConfig
	Storage  StorageConfig
	Payment  PaymentConfig
	Sentry   SentryConfig
}

type ServerConfig struct {
	Env  string
	Port string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	// Upstash REST API (primary method)
	UpstashRestURL   string
	UpstashRestToken string
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

type SMSConfig struct {
	Username string
	APIKey   string
	SenderID string
}

type StorageConfig struct {
	R2AccountID        string
	R2AccessKeyID      string
	R2SecretAccessKey  string
	R2BucketName       string
	R2PublicURL        string
}

type PaymentConfig struct {
	PaystackSecretKey      string
	PaystackPublicKey      string
	FlutterwaveSecretKey   string
	FlutterwavePublicKey   string
	FlutterwaveEncryptKey  string
}

type SentryConfig struct {
	DSN string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	// Parse JWT expiration
	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRES_IN format: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Env:  getEnv("GO_ENV", "development"),
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			UpstashRestURL:   getEnv("UPSTASH_REDIS_REST_URL", ""),
			UpstashRestToken: getEnv("UPSTASH_REDIS_REST_TOKEN", ""),
		},
		JWT: JWTConfig{
			Secret:    getEnv("JWT_SECRET", ""),
			ExpiresIn: jwtExpiry,
		},
		SMS: SMSConfig{
			Username: getEnv("AT_USERNAME", ""),
			APIKey:   getEnv("AT_API_KEY", ""),
			SenderID: getEnv("AT_SENDER_ID", ""),
		},
		Storage: StorageConfig{
			R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
			R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		},
		Payment: PaymentConfig{
			PaystackSecretKey:     getEnv("PAYSTACK_SECRET_KEY", ""),
			PaystackPublicKey:     getEnv("PAYSTACK_PUBLIC_KEY", ""),
			FlutterwaveSecretKey:  getEnv("FLUTTERWAVE_SECRET_KEY", ""),
			FlutterwavePublicKey:  getEnv("FLUTTERWAVE_PUBLIC_KEY", ""),
			FlutterwaveEncryptKey: getEnv("FLUTTERWAVE_ENCRYPTION_KEY", ""),
		},
		Sentry: SentryConfig{
			DSN: getEnv("SENTRY_DSN", ""),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required configuration values are set
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
