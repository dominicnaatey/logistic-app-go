# Developer Guide
**Cross-Border Trucking Logistics Platform - Go Backend**

Last Updated: 2026-08-11 | Phase: 0 (Foundation) Complete

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Getting Started](#getting-started)
3. [Project Structure](#project-structure)
4. [Architecture & Design Principles](#architecture--design-principles)
5. [Development Workflow](#development-workflow)
6. [Database & Migrations](#database--migrations)
7. [Testing Strategy](#testing-strategy)
8. [Deployment](#deployment)
9. [Troubleshooting](#troubleshooting)

---

## Project Overview

### What We're Building
A high-performance backend for a cross-border freight marketplace connecting shippers with truck fleets and independent operators across Mali ↔ Ghana. Think "Uber for long-haul trucking" with smart instant matching, live GPS tracking, and mobile money payments.

### Why Go?
Migrating from NestJS to Go for:
- **10x faster** request throughput
- **5x lower** memory usage
- **Better concurrency** for real-time GPS tracking
- **Simpler deployment** (single binary)

### Tech Stack
| Component | Technology | Purpose |
|---|---|---|
| Framework | Gin | HTTP routing and middleware |
| Database | PostgreSQL + PostGIS | Relational data + geospatial queries |
| Cache | Redis | OTP storage, live location cache, pub/sub |
| Queue | asynq | Background jobs (matching, notifications) |
| Storage | Cloudflare R2 | KYC docs, customs documents |
| SMS | Africa's Talking | OTP delivery |
| Payments | Paystack + Flutterwave | Mobile money, cross-border |

---

## Getting Started

### Prerequisites
```bash
# Required
- Go 1.22+
- Docker Desktop (for PostgreSQL + Redis)
- Git

# Optional (for full functionality)
- PostgreSQL 15+ with PostGIS
- Redis 7+
- Africa's Talking API account
- Paystack/Flutterwave accounts
```

### First-Time Setup

1. **Clone the repository**
   ```bash
   git clone <repo-url>
   cd logistic-app-go
   ```

2. **Install dependencies**
   ```bash
   go mod download
   # Or use: make install
   ```

3. **Set up environment**
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

4. **Start local services**
   ```bash
   # Start PostgreSQL + Redis
   docker-compose up -d
   
   # Wait for services to be ready (10-15 seconds)
   # Then verify connections:
   go run cmd/check/main.go
   ```

5. **Run the server**
   ```bash
   go run cmd/server/main.go
   # Server starts on http://localhost:8080
   ```

6. **Test the API**
   ```bash
   curl http://localhost:8080/health
   # Should return: {"success":true,"data":{"status":"healthy","env":"development"}}
   ```

### Quick Commands (Makefile)
```bash
make help          # Show all available commands
make dev           # Run development server
make check         # Run health checks
make test          # Run tests
make build         # Build binaries
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
```

---

## Project Structure

### Directory Layout
```
logistic-app-go/
├── cmd/                        # Application entry points
│   ├── server/main.go          # HTTP server
│   ├── worker/main.go          # Background job processor (Phase 4+)
│   └── check/main.go           # Health check CLI
│
├── internal/                   # Private application code (Phase 1+)
│   ├── auth/                   # Authentication (OTP, JWT, middleware)
│   ├── user/                   # User management
│   ├── fleet/                  # Fleet companies & trucks
│   ├── driver/                 # Driver onboarding & KYC
│   ├── load/                   # Load posting & pricing
│   ├── trip/                   # Trip lifecycle & tracking
│   ├── payment/                # Escrow & payouts
│   ├── location/               # GPS tracking & WebSocket
│   ├── matching/               # Truck matching algorithm
│   ├── admin/                  # Admin dashboard backend
│   ├── sms/                    # SMS service wrapper
│   └── storage/                # File storage (R2/S3)
│
├── pkg/                        # Public, reusable packages
│   ├── config/                 # Configuration management
│   ├── database/               # Database connection
│   ├── cache/                  # Redis client
│   ├── queue/                  # Job queue (Phase 4+)
│   └── response/               # HTTP response helpers
│
├── docs/                       # Documentation
│   ├── DEVELOPER-GUIDE.md      # This file
│   ├── API.md                  # API documentation (Phase 1+)
│   ├── ARCHITECTURE.md         # System design (Phase 1+)
│   └── DEPLOYMENT.md           # Deployment guide (Phase 10+)
│
├── migrations/                 # Database migrations (Phase 1+)
│   ├── 001_initial_schema.sql
│   └── ...
│
├── .env.example                # Environment template
├── docker-compose.yml          # Local development services
├── Makefile                    # Common commands
└── README.md                   # Project overview
```

### Package Organization Principles

**1. `cmd/` - Application Entry Points**
- Each subdirectory is a separate executable
- Minimal logic — delegates to `internal/` and `pkg/`
- Example: `cmd/server/main.go` wires up dependencies and starts HTTP server

**2. `internal/` - Private Business Logic**
- NOT importable by external projects
- Domain-driven structure (auth, fleet, trip, etc.)
- Each domain has: `handler.go`, `service.go`, `repository.go`, `dto.go`

**3. `pkg/` - Public Utilities**
- Reusable, domain-agnostic packages
- Could be extracted into separate libraries
- No dependencies on `internal/`

---

## Architecture & Design Principles

### Layered Architecture

```
HTTP Request
    ↓
Handler (controllers)          # Validate input, call service
    ↓
Service (business logic)       # Orchestrate operations, enforce rules
    ↓
Repository (data access)       # CRUD operations, queries
    ↓
Database / External API
```

### Key Design Patterns

**1. Dependency Injection**
```go
// Services receive dependencies via constructor
func NewAuthService(
    otpService OTPService,
    jwtService JWTService,
    smsService SMSService,
    userRepo UserRepository,
) AuthService {
    return &authService{
        otpService: otpService,
        jwtService: jwtService,
        smsService: smsService,
        userRepo:   userRepo,
    }
}
```

**2. Interface-Based Design (Modularity)**
```go
// Define interfaces, not concrete types
type SMSService interface {
    SendOTP(phone, code string) error
}

// Easy to swap implementations (Africa's Talking → Twilio)
type africastalkingSMS struct { /* ... */ }
type twilioSMS struct { /* ... */ }
```

**3. Repository Pattern**
```go
// Abstract database access
type UserRepository interface {
    FindByID(id uuid.UUID) (*User, error)
    FindByPhone(phone string) (*User, error)
    Create(user *User) error
    Update(user *User) error
}

// Implementation uses GORM, but could swap to sqlx or raw SQL
```

**4. Context Propagation**
```go
// Always pass context for cancellation and deadlines
func (s *authService) VerifyOTP(ctx context.Context, phone, code string) error {
    // Check Redis with context timeout
    storedCode, err := s.cache.Get(ctx, fmt.Sprintf("otp:%s", phone))
    // ...
}
```

### Modularity Guidelines

**✅ DO:**
- Define interfaces for all external dependencies (SMS, payment, storage)
- Use constructor injection (not global variables)
- Keep packages focused (single responsibility)
- Write tests using mock implementations

**❌ DON'T:**
- Import `internal/` packages from other domains (use interfaces)
- Use global state (except config and logger)
- Hardcode external URLs or credentials
- Tight-couple to third-party libraries

---

## Development Workflow

### Adding a New Feature (Example: Driver Rating)

1. **Define the interface** (`internal/rating/service.go`)
   ```go
   type RatingService interface {
       CreateRating(ctx context.Context, req CreateRatingRequest) error
       GetDriverRating(ctx context.Context, driverID uuid.UUID) (float64, error)
   }
   ```

2. **Create the repository** (`internal/rating/repository.go`)
   ```go
   type RatingRepository interface {
       Create(ctx context.Context, rating *Rating) error
       FindByDriverID(ctx context.Context, driverID uuid.UUID) ([]Rating, error)
   }
   ```

3. **Implement the service** (`internal/rating/service.go`)
   ```go
   type ratingService struct {
       repo RatingRepository
   }
   
   func (s *ratingService) CreateRating(ctx context.Context, req CreateRatingRequest) error {
       // Business logic here
   }
   ```

4. **Add HTTP handlers** (`internal/rating/handler.go`)
   ```go
   func (h *RatingHandler) CreateRating(c *gin.Context) {
       // Parse request, call service, return response
   }
   ```

5. **Wire up in main** (`cmd/server/main.go`)
   ```go
   ratingRepo := rating.NewRepository(db)
   ratingService := rating.NewService(ratingRepo)
   ratingHandler := rating.NewHandler(ratingService)
   
   v1.POST("/ratings", ratingHandler.CreateRating)
   ```

6. **Write tests** (`internal/rating/service_test.go`)
   ```go
   func TestCreateRating(t *testing.T) {
       mockRepo := &MockRatingRepository{}
       service := NewService(mockRepo)
       // Test logic
   }
   ```

### Code Style Guidelines

**Naming Conventions:**
- Packages: lowercase, single word (`auth`, not `authService`)
- Interfaces: noun or verb phrase (`UserRepository`, `SMSService`)
- Structs: PascalCase (`FleetCompany`, `TripStop`)
- Functions: PascalCase for exported, camelCase for private
- Constants: PascalCase or UPPER_SNAKE for environment vars

**Comments:**
```go
// Every exported function/type needs a doc comment

// NewAuthService creates a new authentication service with the provided dependencies.
// It returns an AuthService implementation that handles OTP verification and JWT issuance.
func NewAuthService(/* ... */) AuthService {
    // ...
}
```

**Error Handling:**
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// Log errors before returning 500s
if err := service.DoSomething(ctx); err != nil {
    log.Printf("ERROR: DoSomething failed: %v", err)
    response.InternalError(c, "Operation failed")
    return
}
```

---

## Database & Migrations

### Schema Management (Phase 1+)

We use `golang-migrate/migrate` for database migrations.

**Create a migration:**
```bash
migrate create -ext sql -dir migrations -seq add_users_table
# Creates:
# migrations/000001_add_users_table.up.sql
# migrations/000001_add_users_table.down.sql
```

**Run migrations:**
```bash
migrate -path migrations -database "postgresql://user:pass@localhost:5432/db?sslmode=disable" up
```

### PostGIS Usage

**Store locations as geography points:**
```sql
CREATE TABLE trucks (
    id UUID PRIMARY KEY,
    current_location GEOGRAPHY(POINT, 4326),
    -- ...
);
```

**Query nearest trucks:**
```sql
SELECT id, ST_Distance(current_location, ST_MakePoint(-0.1870, 5.6037)::geography) AS distance_meters
FROM trucks
WHERE status = 'available'
ORDER BY distance_meters
LIMIT 10;
```

**GORM representation:**
```go
type Truck struct {
    ID              uuid.UUID
    CurrentLocation string `gorm:"type:geography(Point,4326)"` // Stored as WKT or GeoJSON
}
```

---

## Testing Strategy

### Test Structure
```
internal/auth/
├── handler.go
├── handler_test.go       # HTTP handler tests (integration)
├── service.go
├── service_test.go       # Business logic tests (unit)
├── repository.go
└── repository_test.go    # Database tests (integration)
```

### Unit Tests (Business Logic)
```go
func TestVerifyOTP_Success(t *testing.T) {
    // Arrange: mock dependencies
    mockCache := &MockCache{}
    mockCache.On("Get", mock.Anything, "otp:+233201234567").Return("123456", nil)
    
    service := NewOTPService(mockCache)
    
    // Act
    err := service.VerifyOTP(context.Background(), "+233201234567", "123456")
    
    // Assert
    assert.NoError(t, err)
    mockCache.AssertExpectations(t)
}
```

### Integration Tests (HTTP Endpoints)
```go
func TestCreateLoad_Integration(t *testing.T) {
    // Set up test database
    db := setupTestDB(t)
    defer teardownTestDB(t, db)
    
    // Create test server
    router := setupTestRouter(db)
    w := httptest.NewRecorder()
    
    // Make request
    req, _ := http.NewRequest("POST", "/api/v1/loads", bytes.NewBuffer(payload))
    router.ServeHTTP(w, req)
    
    // Assert response
    assert.Equal(t, 201, w.Code)
}
```

### Running Tests
```bash
# All tests
go test ./...

# Specific package
go test ./internal/auth/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

---

## Deployment

*(This section will be expanded in Phase 10)*

### Environment-Specific Configuration

**Development:**
- Local PostgreSQL + Redis via Docker
- Debug logging enabled
- Hot reload (using `air` or similar)

**Staging:**
- Managed PostgreSQL (Neon)
- Managed Redis (Upstash)
- Test API keys for payments/SMS

**Production:**
- AWS RDS PostgreSQL + ElastiCache Redis
- Production API keys
- Error tracking (Sentry)
- Monitoring (CloudWatch)

---

## Troubleshooting

### Common Issues

**1. "Failed to connect to database"**
- Check Docker is running: `docker ps`
- Verify DATABASE_URL in `.env`
- Run health check: `go run cmd/check/main.go`

**2. "Failed to ping Redis"**
- Check Redis is running: `docker ps | grep redis`
- Test manually: `redis-cli ping`

**3. "PostGIS version query failed"**
- PostGIS not installed: `docker exec -it logistic_postgres psql -U logistic_user -d logistic_app -c "CREATE EXTENSION postgis;"`

**4. Build errors**
- Run `go mod tidy` to sync dependencies
- Clear cache: `go clean -modcache`

### Getting Help

- Check this guide first
- Review phase completion docs (`PHASE-X-COMPLETE.md`)
- Check implementation plan (`IMPLEMENTATION-PLAN.md`)
- Search project issues (when using issue tracker)

---

## Contributing Guidelines

*(Will be expanded as team grows)*

**Before submitting a PR:**
1. Run tests: `make test`
2. Run linter: `golangci-lint run`
3. Update documentation if adding features
4. Follow commit message format: `type(scope): description`

---

**Last Updated:** 2026-08-11 (Phase 0)
**Next Update:** Phase 1 completion (add auth documentation)
