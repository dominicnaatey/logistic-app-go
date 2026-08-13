# Developer Guide
**Cross-Border Trucking Logistics Platform — Go Backend**

Last Updated: 2026-08-13 | Phase 1 (Authentication) Complete

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Getting Started](#2-getting-started)
3. [Project Structure](#3-project-structure)
4. [Architecture & Design Principles](#4-architecture--design-principles)
5. [Authentication System](#5-authentication-system)
6. [Development Workflow](#6-development-workflow)
7. [Database & Migrations](#7-database--migrations)
8. [Testing Strategy](#8-testing-strategy)
9. [API Reference](#9-api-reference)
10. [Deployment](#10-deployment)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Project Overview

### What We're Building
A high-performance backend for a cross-border freight marketplace connecting shippers with truck fleets and independent operators across Mali ↔ Ghana. Think "Uber for long-haul trucking" — smart instant matching, live GPS tracking, mobile money payments, and offline-first mobile apps.

### Why Go?
Migrated from NestJS for:
- **10x faster** request throughput
- **5x lower** memory usage
- **Better concurrency** for real-time GPS tracking
- **Single binary** deployment

### Tech Stack
| Component | Technology | Purpose |
|-----------|-----------|---------|
| Framework | Gin | HTTP routing and middleware |
| Database | Neon PostgreSQL + PostGIS | Relational data + geospatial queries |
| Cache | Upstash Redis (REST) | OTP storage, rate limiting, live location cache |
| Storage | Cloudflare R2 | KYC docs, customs documents, proof of delivery |
| SMS | Africa's Talking | OTP delivery, trip notifications |
| Auth | JWT (HS256) + OTP | Phone-based, stateless authentication |
| Payments | Paystack + Flutterwave | Mobile money, cross-border (Phase 3+) |

---

## 2. Getting Started

### Prerequisites
```bash
# Required
- Go 1.22+
- A Neon PostgreSQL account (free at https://console.neon.tech)
- An Upstash Redis account (free at https://console.upstash.com)
- A Cloudflare R2 bucket
- Africa's Talking API account (sandbox is free)
- Git
```

### First-Time Setup

**1. Clone and install dependencies**
```bash
git clone https://github.com/dominicnaatey/logistic-app-go.git
cd logistic-app-go
git checkout dev   # all active development is here
go mod download
```

**2. Configure environment**
```bash
cp .env.example .env
# Open .env and fill in your credentials
```

Required `.env` values:
```env
DATABASE_URL=postgresql://...         # Neon connection string
UPSTASH_REDIS_REST_URL=https://...
UPSTASH_REDIS_REST_TOKEN=...
JWT_SECRET=...                        # Generate with: go run cmd/keygen/main.go
AT_USERNAME=sandbox
AT_API_KEY=...                        # Africa's Talking API key
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=trucking-logistics-app
```

**3. Verify all connections**
```bash
go run cmd/check/main.go
# Expected:
# ✓ PostgreSQL connected (PostGIS 3.6)
# ✓ Upstash Redis connected
# ✓ JWT configuration valid
# ✓ R2 bucket accessible
```

**4. Start the server**
```bash
go run cmd/server/main.go
# Server starts on http://localhost:8080
# Migrations run automatically on startup
```

**5. Test the API**
```bash
curl http://localhost:8080/health
# {"success":true,"data":{"status":"healthy","env":"development"}}

curl -X POST http://localhost:8080/api/v1/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{"phone":"+233501234567","role":"driver","language":"en"}'
```

### Quick Commands (Makefile)
```bash
make help       # Show all commands
make dev        # Run development server
make check      # Run health checks
make build      # Build all binaries
make docker-up  # Start local Docker services
```

---

## 3. Project Structure

```
logistic-app-go/
├── cmd/                            # Application entry points
│   ├── server/main.go              # HTTP server (wires all dependencies)
│   ├── check/main.go               # Health check CLI tool
│   ├── keygen/main.go              # Secure JWT key generator
│   ├── test_user/main.go           # User model integration test
│   ├── test_otp/main.go            # OTP service integration test
│   ├── test_sms/main.go            # SMS service integration test
│   ├── test_jwt_middleware/main.go # JWT middleware integration test
│   └── test_auth/main.go           # Full auth flow end-to-end test
│
├── internal/                       # Private business logic (not importable externally)
│   ├── auth/
│   │   ├── otp_service.go          # OTP generate/store/verify/rate-limit
│   │   ├── middleware.go           # JWTMiddleware, RequireRoles, GetClaims
│   │   └── handler.go              # HTTP handlers: send-otp, verify-otp, me
│   ├── sms/
│   │   └── service.go              # Africa's Talking SMS wrapper
│   └── user/
│       ├── model.go                # User GORM struct, role/KYC constants
│       ├── repository.go           # Database CRUD (interface + GORM impl)
│       └── service.go              # FindOrCreate, GetByID, UpdateProfile
│
├── pkg/                            # Shared, reusable packages
│   ├── auth/jwt.go                 # JWT manager (generate, verify, refresh)
│   ├── cache/upstash.go            # Upstash Redis REST client
│   ├── config/config.go            # Config struct + env loader + validation
│   ├── database/
│   │   ├── postgres.go             # PostgreSQL + PostGIS connection
│   │   └── migrate.go              # GORM auto-migration runner
│   ├── response/response.go        # JSON response helpers
│   └── storage/r2.go               # Cloudflare R2 S3-compatible client
│
├── migrations/
│   └── 001_create_users.sql        # Raw SQL migration (reference/backup)
│
├── docs/                           # All documentation lives here
├── .env                            # Your secrets (NEVER COMMIT)
├── .env.example                    # Template — safe to commit
├── docker-compose.yml              # Local PostgreSQL + Redis
├── Makefile                        # Common dev commands
└── go.mod / go.sum                 # Dependency management
```

### Package Organisation Rules

| Package | What Lives Here | Rule |
|---------|----------------|------|
| `cmd/` | `main.go` files, test runners | Wire dependencies, no business logic |
| `internal/` | Domain logic (auth, user, fleet…) | Only this project can import it |
| `pkg/` | Reusable utilities (config, db, cache) | No dependency on `internal/` |
| `migrations/` | Raw `.sql` files | One file per schema change, numbered sequentially |

---

## 4. Architecture & Design Principles

### Layered Architecture
```
HTTP Request
    ↓
Handler             # Parse + validate input, call service, format response
    ↓
Service             # Business rules, orchestration, error handling
    ↓
Repository          # Database queries, data mapping
    ↓
PostgreSQL / Redis / R2 / AT
```

### Interface-Based Design (Core Principle)

Every service dependency is an **interface**, not a concrete type. This makes services:
- Testable (swap real with mock in tests)
- Decoupled (swap Africa's Talking with Twilio without touching business logic)
- Clear (interface is the contract, implementation is the detail)

```go
// GOOD — depend on the interface
type SMSService interface {
    SendOTP(phone, code string) error
    SendWelcome(phone, name, lang string) error
}

// BAD — depend on the concrete type
type AuthHandler struct {
    sms *sms.Service  // tightly coupled to AT implementation
}
```

### Dependency Injection Pattern

All dependencies are wired in `cmd/server/main.go`. No global variables.

```go
// cmd/server/main.go
db        := database.NewPostgresDB(cfg.Database.URL, isDev)
redis     := cache.NewUpstashClient(cfg.Redis.URL, cfg.Redis.Token)
jwtMgr    := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
smsSvc    := sms.NewService(cfg.SMS.Username, cfg.SMS.APIKey, cfg.SMS.SenderID)
userRepo  := user.NewRepository(db)
userSvc   := user.NewService(userRepo)
otpSvc    := internalAuth.NewOTPService(redis)
authHdlr  := internalAuth.NewHandler(otpSvc, smsSvc, userSvc, jwtMgr)
```

### Repository Pattern

Repositories isolate database access. The service layer never touches GORM directly.

```go
// Interface (in model.go or repository.go)
type Repository interface {
    FindByPhone(phone string) (*User, error)
    FindByID(id uuid.UUID) (*User, error)
    Create(u *User) error
    Update(u *User) error
}

// GORM implementation
type gormRepository struct { db *gorm.DB }

func (r *gormRepository) FindByPhone(phone string) (*User, error) {
    var u User
    err := r.db.Where("phone = ?", phone).First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil  // not found is not an error at this layer
    }
    return &u, err
}
```

---

## 5. Authentication System

### Overview

Authentication is phone-based using OTP (One-Time Password). No passwords.

```
Phone → OTP SMS → Verify → JWT Token
```

### User Roles

| Role | Description | Can Accept Loads? |
|------|-------------|------------------|
| `shipper` | Posts loads for transport | No |
| `driver` | Drives for a fleet company | Yes (if KYC approved) |
| `fleet_admin` | Manages fleet company | No |
| `owner_operator` | Independent truck owner-driver | Yes (if KYC approved) |
| `admin` | Platform administrator | No |

### KYC Status

| Status | Meaning |
|--------|---------|
| `pending` | Default on registration, awaiting documents |
| `approved` | Identity verified — driver can accept loads |
| `rejected` | Documents rejected — must re-submit |

### OTP Flow

```
Client                  Backend                 Redis / AT
  |                        |                        |
  |-- POST /send-otp ----→ |                        |
  |   {phone, role}        |-- GenerateCode ------→ |
  |                        |-- SET otp:{phone} ---→ | TTL: 10 min
  |                        |-- INCR rate:{phone} → | TTL: 10 min (max 3)
  |                        |-- SendSMS ----------→ Africa's Talking
  |← {message: "sent"} --- |                        |
  |                        |                        |
  |-- POST /verify-otp --→ |                        |
  |   {phone, code}        |-- GET otp:{phone} ---→ |
  |                        |-- Compare codes        |
  |                        |-- DEL otp:{phone} ---→ | (single-use)
  |                        |-- FindOrCreate user    |
  |                        |-- Issue JWT            |
  |← {token, is_new, user} |                        |
```

### JWT Claims Structure

```go
type Claims struct {
    UserID uuid.UUID `json:"sub"`    // User's database UUID
    Phone  string    `json:"phone"`  // E.164 phone number
    Role   string    `json:"role"`   // One of the role constants
    jwt.RegisteredClaims             // exp, iat, nbf
}
```

### Protecting Routes

```go
// In setupRouter (cmd/server/main.go):

// Public (no JWT)
v1.POST("/auth/send-otp", authHandler.SendOTP)
v1.POST("/auth/verify-otp", authHandler.VerifyOTP)

// JWT required
protected := v1.Group("/", internalAuth.JWTMiddleware(jwtManager))
protected.GET("/auth/me", authHandler.Me)

// Role-restricted
adminOnly := protected.Group("/", internalAuth.RequireRoles(user.RoleAdmin))
adminOnly.GET("/admin/users", adminHandler.ListUsers)

// Multiple roles
driverRoutes := protected.Group("/", internalAuth.RequireRoles(
    user.RoleDriver, user.RoleOwnerOperator,
))
driverRoutes.POST("/trips/:id/accept", tripHandler.Accept)
```

### Reading the Authenticated User in a Handler

```go
func (h *MyHandler) SomeProtectedRoute(c *gin.Context) {
    claims := internalAuth.GetClaims(c)
    // claims.UserID  — uuid.UUID
    // claims.Phone   — string
    // claims.Role    — string

    user, err := h.userSvc.GetByID(claims.UserID)
    // ...
}
```

### Generating a JWT Secret

```bash
go run cmd/keygen/main.go
# Output: a 44-character base64-encoded 256-bit key
# Paste it into .env as JWT_SECRET
```

---

## 6. Development Workflow

### Branch Strategy

```
main      ← Production-ready, tagged releases
  └── dev ← Integration branch, all features merge here
       ├── feature/fleet-management
       ├── feature/load-posting
       └── feature/matching-engine
```

### Adding a New Domain (Example: Phase 2 Fleet)

**1. Create the model** (`internal/fleet/model.go`)
```go
type FleetCompany struct {
    ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
    BusinessName string    `gorm:"not null"`
    // ...
}

func (f *FleetCompany) BeforeCreate(tx *gorm.DB) error {
    if f.ID == uuid.Nil { f.ID = uuid.New() }
    return nil
}
```

**2. Create the repository** (`internal/fleet/repository.go`)
```go
type Repository interface {
    FindByID(id uuid.UUID) (*FleetCompany, error)
    Create(f *FleetCompany) error
}

type gormRepository struct{ db *gorm.DB }
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db} }
```

**3. Create the service** (`internal/fleet/service.go`)
```go
type Service interface {
    Register(ownerID uuid.UUID, name string) (*FleetCompany, error)
}

type service struct{ repo Repository }
func NewService(repo Repository) Service { return &service{repo} }
```

**4. Create handlers** (`internal/fleet/handler.go`)
```go
type Handler struct{ svc Service }
func NewHandler(svc Service) *Handler { return &Handler{svc} }

func (h *Handler) Register(c *gin.Context) { /* ... */ }
func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
    protected.POST("/fleet/register", h.Register)
}
```

**5. Add migration** (`pkg/database/migrate.go`)
```go
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &user.User{},
        &fleet.FleetCompany{},  // add here
    )
}
```

**6. Wire in server** (`cmd/server/main.go`)
```go
fleetRepo := fleet.NewRepository(db)
fleetSvc  := fleet.NewService(fleetRepo)
fleetHdlr := fleet.NewHandler(fleetSvc)
fleetHdlr.RegisterRoutes(protected)
```

### Commit Message Convention

```
feat(auth):     Add JWT refresh endpoint
fix(otp):       Correct rate limit counter TTL
docs(api):      Add /me endpoint documentation
refactor(user): Extract phone validation helper
test(fleet):    Add fleet registration integration test
chore(deps):    Upgrade gin to v1.12.1
```

### Code Style

**Error handling** — always wrap with context:
```go
if err != nil {
    return fmt.Errorf("FindOrCreate lookup: %w", err)
}
```

**Godoc comments** — all exported identifiers:
```go
// FindOrCreate looks up a user by phone or creates a new one on first login.
// Returns (user, isNew, error). isNew=true on first-ever login.
func (s *service) FindOrCreate(phone, role, language string) (*User, bool, error) {
```

**Response helpers** — never write raw `c.JSON` in handlers:
```go
response.Success(c, data)          // 200
response.Created(c, data)          // 201
response.BadRequest(c, "message")  // 400
response.Unauthorized(c, "msg")    // 401
response.Forbidden(c, "message")   // 403
response.NotFound(c, "message")    // 404
response.InternalError(c, "msg")   // 500
```

---

## 7. Database & Migrations

### Auto-Migration

Migrations run automatically every time the server starts:

```go
// pkg/database/migrate.go
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &user.User{},
        // Add new models here as phases progress
    )
}
```

GORM auto-migration:
- Creates tables that don't exist
- Adds missing columns
- Creates missing indexes
- **Never drops columns or tables** (safe to run on every start)

### Adding a Migration

1. Add the GORM model struct in `internal/<domain>/model.go`
2. Register it in `pkg/database/migrate.go`
3. Optionally add a raw `.sql` file in `migrations/` for reference

### PostGIS Usage

```go
// Store as geography point (WGS84)
type Truck struct {
    CurrentLocation string `gorm:"type:geography(Point,4326)"`
}

// Raw spatial query — find trucks within 50km
db.Raw(`
    SELECT id, registration,
           ST_Distance(current_location::geography,
                       ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance_m
    FROM trucks
    WHERE ST_DWithin(
        current_location::geography,
        ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
        50000
    )
    ORDER BY distance_m
`, lng, lat, lng, lat).Scan(&results)
```

### Soft Delete

All models use GORM soft delete via `gorm.DeletedAt`:

```go
// Soft delete (sets deleted_at timestamp)
db.Delete(&user)

// Query — GORM automatically filters WHERE deleted_at IS NULL
db.Find(&users)

// Include soft-deleted records
db.Unscoped().Find(&users)
```

---

## 8. Testing Strategy

### Integration Test Pattern

All tests in `cmd/test_*/main.go` follow this pattern:
1. Connect to live infrastructure (Neon, Upstash, AT)
2. Clean up any leftover state from previous runs
3. Run test scenarios
4. Clean up test data
5. Print summary

```bash
# Run all integration tests
go run cmd/test_user/main.go
go run cmd/test_otp/main.go
go run cmd/test_sms/main.go
go run cmd/test_jwt_middleware/main.go
go run cmd/test_auth/main.go
```

### In-Process HTTP Testing (No Network)

JWT middleware and handler tests use `httptest` — no server needed:

```go
// Spin up a real Gin router in memory
router := buildTestRouter(jwtManager)

// Fire a request
req := httptest.NewRequest("POST", "/auth/send-otp", body)
w   := httptest.NewRecorder()
router.ServeHTTP(w, req)

assert.Equal(t, 200, w.Code)
```

### Unit Test Pattern (for future use)

```go
// Mock the repository
type mockUserRepo struct {
    findByPhone func(phone string) (*user.User, error)
}
func (m *mockUserRepo) FindByPhone(p string) (*user.User, error) { return m.findByPhone(p) }

// Test the service in isolation
svc := user.NewService(&mockUserRepo{
    findByPhone: func(p string) (*user.User, error) { return nil, nil },
})
u, isNew, err := svc.FindOrCreate("+233501234567", user.RoleDriver, "en")
assert.True(t, isNew)
```

---

## 9. API Reference

### Authentication

#### POST /api/v1/auth/send-otp
Request OTP code via SMS.

```json
// Request
{
    "phone": "+233501234567",   // required, E.164 format
    "role": "driver",           // required: shipper|driver|fleet_admin|owner_operator
    "language": "en"            // optional: "en" (default) or "fr"
}

// Response 200
{
    "success": true,
    "data": { "message": "OTP sent successfully", "phone": "+233501234567" }
}

// Response 429 (rate limited)
{
    "success": false,
    "error": "too many OTP requests — maximum 3 per 10m0s, please try again later"
}
```

#### POST /api/v1/auth/verify-otp
Verify OTP and receive JWT. Pass `role` and `language` as query params.

```
POST /api/v1/auth/verify-otp?role=driver&language=en
```

```json
// Request
{
    "phone": "+233501234567",
    "code": "123456"
}

// Response 201 (new user created)
{
    "success": true,
    "token": "eyJhbGci...",
    "is_new": true,
    "user": {
        "id": "uuid",
        "phone": "+233501234567",
        "role": "driver",
        "language": "en",
        "kyc_status": "pending"
    }
}

// Response 200 (existing user)
// Same shape, is_new: false

// Response 401 (wrong/expired code)
{ "success": false, "error": "invalid OTP code" }
```

#### GET /api/v1/auth/me
Get authenticated user's profile. Requires `Authorization: Bearer <token>`.

```json
// Response 200
{
    "success": true,
    "data": {
        "id": "uuid",
        "phone": "+233501234567",
        "role": "driver",
        "name": "Kwame Mensah",
        "language": "en",
        "kyc_status": "pending",
        "is_active": true,
        "created_at": "2026-08-13T...",
        "updated_at": "2026-08-13T..."
    }
}
```

#### GET /health
```json
{ "success": true, "data": { "status": "healthy", "env": "development" } }
```

---

## 10. Deployment

*Full deployment guide will be added at Phase 10. Summary below.*

### Environments

| Environment | Database | Redis | Mode |
|-------------|----------|-------|------|
| Development | Neon (dev branch) | Upstash | `GO_ENV=development` |
| Staging | Neon (staging branch) | Upstash | `GO_ENV=staging` |
| Production | Neon (main branch) | Upstash | `GO_ENV=production` |

### Build

```bash
# Build server binary
go build -o bin/server cmd/server/main.go

# Cross-compile for Linux (common deployment target)
GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/server/main.go
```

### Environment Variables Required in Production

All variables from `.env.example` — set as environment variables on your hosting platform, never as files.

---

## 11. Troubleshooting

### "Failed to connect to database"
```bash
go run cmd/check/main.go  # diagnose which service is failing
# Verify DATABASE_URL in .env
# Check Neon dashboard for connection limits
```

### "OTP expired or not found"
- OTPs expire after 10 minutes
- Check Upstash dashboard to see if key exists: `otp:+233...`
- Rate limit: max 3 per phone per 10 minutes — wait for TTL to expire

### "Invalid or expired token"
- JWT secret may have changed (all existing tokens invalidated)
- Check token expiry (`JWT_EXPIRES_IN` — default 168h)
- Decode token at jwt.io to inspect claims

### "SMS not received"
- In sandbox: check AT simulator at https://simulator.africastalking.com
- In production: verify `AT_API_KEY` is the production key, not sandbox

### Build errors
```bash
go mod tidy       # sync dependencies
go build ./...    # show all compile errors
```

### PostGIS query failing
```bash
# Verify PostGIS is enabled
go run cmd/check/main.go
# Should show: ✓ PostGIS version: 3.6
```

---

## Related Documentation

- [Architecture Overview](ARCHITECTURE.md) — System design and decisions
- [JWT Setup Guide](JWT-SETUP.md) — Authentication deep-dive
- [Git Workflow](GIT-WORKFLOW.md) — Branching strategy
- [Quick Reference](QUICK-REFERENCE.md) — Commands and patterns
- [Project Context Log](PROJECT-CONTEXT-LOG.md) — Full history for AI handoff

---

**Last Updated**: 2026-08-13  
**Current Phase**: Phase 1 (Authentication) Complete  
**Next Phase**: Phase 2 — Core Domain Models (Fleet, Trucks, Drivers)
