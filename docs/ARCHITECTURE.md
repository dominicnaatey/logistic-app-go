# Architecture Overview
**Cross-Border Trucking Logistics Platform**

Last Updated: 2026-08-11 | Current Phase: 0 (Foundation)

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Mobile Applications                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Shipper App  │  │  Driver App  │  │  Fleet Portal │      │
│  │ (React Native)│  │(React Native)│  │  (Web/RN)    │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          │    REST API      │    REST API      │    REST API
          │    WebSocket     │    WebSocket     │
          └──────────────────┼──────────────────┘
                             │
┌────────────────────────────▼───────────────────────────────┐
│                   API Gateway / Load Balancer               │
│                        (Future: AWS ALB)                    │
└────────────────────────────┬───────────────────────────────┘
                             │
┌────────────────────────────▼───────────────────────────────┐
│                   Go/Gin Backend Server                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  HTTP Handlers (Gin)                                │   │
│  │  ├─ Auth (OTP, JWT)                                 │   │
│  │  ├─ Loads (CRUD, matching trigger)                  │   │
│  │  ├─ Trips (tracking, status updates)                │   │
│  │  ├─ Payments (escrow, payouts)                      │   │
│  │  └─ Admin (KYC, disputes)                           │   │
│  └──────────────────┬──────────────────────────────────┘   │
│  ┌─────────────────▼──────────────────────────────────┐   │
│  │  Business Logic Services                            │   │
│  │  ├─ Matching Engine (PostGIS queries)              │   │
│  │  ├─ Pricing Calculator                             │   │
│  │  ├─ Payment Orchestration                          │   │
│  │  └─ Notification Dispatcher                        │   │
│  └──────────────────┬──────────────────────────────────┘   │
│  ┌─────────────────▼──────────────────────────────────┐   │
│  │  Repository Layer (Data Access)                     │   │
│  │  ├─ GORM (PostgreSQL ORM)                          │   │
│  │  ├─ Raw SQL (complex PostGIS queries)             │   │
│  │  └─ Redis (cache, pub/sub)                        │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────┘
          │                    │                    │
          ▼                    ▼                    ▼
┌─────────────────┐  ┌──────────────────┐  ┌──────────────┐
│   PostgreSQL    │  │      Redis        │  │  Background  │
│   + PostGIS     │  │  (Cache, Pub/Sub) │  │  Job Queue   │
│  (Persistent    │  │  (Upstash/        │  │  (asynq)     │
│   Storage)      │  │   ElastiCache)    │  │              │
└─────────────────┘  └──────────────────┘  └──────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   External Services                          │
│  ┌───────────────┐  ┌─────────────┐  ┌──────────────────┐  │
│  │ Cloudflare R2 │  │ Paystack/   │  │ Africa's Talking │  │
│  │ (File Storage)│  │ Flutterwave │  │ (SMS Gateway)    │  │
│  └───────────────┘  └─────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. API Layer (Handlers)
**Purpose:** HTTP request handling, input validation, response formatting

**Responsibilities:**
- Parse and validate incoming requests (JSON, query params)
- Call appropriate service methods
- Format responses using standard helpers
- Handle HTTP-specific concerns (status codes, headers)

**Example:**
```go
// internal/auth/handler.go
type AuthHandler struct {
    authService AuthService
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
    var req VerifyOTPRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "Invalid request")
        return
    }
    
    token, err := h.authService.VerifyOTP(c.Request.Context(), req.Phone, req.Code)
    if err != nil {
        response.Unauthorized(c, "Invalid OTP")
        return
    }
    
    response.Success(c, gin.H{"accessToken": token})
}
```

**Decoupling Strategy:**
- Handlers depend ONLY on service interfaces, never concrete types
- Easy to swap service implementations without changing handlers
- Testable via mock services

---

### 2. Service Layer (Business Logic)
**Purpose:** Orchestrate operations, enforce business rules, coordinate between repositories

**Responsibilities:**
- Implement core business logic
- Validate business rules (e.g., "driver can't accept trip if already on active trip")
- Coordinate multiple repositories
- Trigger side effects (notifications, background jobs)

**Example:**
```go
// internal/load/service.go
type LoadService interface {
    CreateLoad(ctx context.Context, req CreateLoadRequest) (*Load, error)
    TriggerMatching(ctx context.Context, loadID uuid.UUID) error
}

type loadService struct {
    loadRepo       LoadRepository
    pricingService PricingService
    matchingQueue  MatchingQueue  // Interface to job queue
}

func (s *loadService) CreateLoad(ctx context.Context, req CreateLoadRequest) (*Load, error) {
    // 1. Calculate price
    price := s.pricingService.Calculate(req.WeightTons, req.Distance, req.CargoType)
    
    // 2. Create load record
    load := &Load{
        ShipperID:  req.ShipperID,
        WeightTons: req.WeightTons,
        TotalPrice: price,
        Status:     "posted",
    }
    
    if err := s.loadRepo.Create(ctx, load); err != nil {
        return nil, fmt.Errorf("failed to create load: %w", err)
    }
    
    // 3. Enqueue matching job
    if err := s.matchingQueue.Enqueue(ctx, load.ID); err != nil {
        log.Printf("Failed to enqueue matching job: %v", err)
        // Don't fail the request — matching can be retried
    }
    
    return load, nil
}
```

**Decoupling Strategy:**
- Services depend on interfaces (repositories, external services)
- No direct database access (use repositories)
- No HTTP-specific code (use context, return errors)
- Easy to test with mocks

---

### 3. Repository Layer (Data Access)
**Purpose:** Abstract database operations, encapsulate queries

**Responsibilities:**
- CRUD operations
- Complex queries (joins, aggregations, PostGIS)
- Transaction management
- No business logic (just data access)

**Example:**
```go
// internal/load/repository.go
type LoadRepository interface {
    Create(ctx context.Context, load *Load) error
    FindByID(ctx context.Context, id uuid.UUID) (*Load, error)
    FindByShipperID(ctx context.Context, shipperID uuid.UUID) ([]Load, error)
    Update(ctx context.Context, load *Load) error
}

type loadRepository struct {
    db *gorm.DB
}

func (r *loadRepository) Create(ctx context.Context, load *Load) error {
    return r.db.WithContext(ctx).Create(load).Error
}

func (r *loadRepository) FindByID(ctx context.Context, id uuid.UUID) (*Load, error) {
    var load Load
    err := r.db.WithContext(ctx).
        Preload("Stops").  // Eager load related stops
        First(&load, "id = ?", id).Error
    
    if err == gorm.ErrRecordNotFound {
        return nil, nil  // Return nil, not error, for not found
    }
    return &load, err
}
```

**Decoupling Strategy:**
- Repository interface defined in same package as domain model
- Implementation can be swapped (GORM → sqlx → raw SQL)
- Easy to create in-memory implementations for tests

---

### 4. External Service Adapters
**Purpose:** Wrap third-party APIs, make them swappable

**Responsibilities:**
- Abstract external service details
- Handle API-specific authentication
- Retry logic, error handling
- Map external data models to our domain models

**Example:**
```go
// internal/sms/service.go
type SMSService interface {
    SendOTP(phone, code string) error
}

// Africa's Talking implementation
type africastalkingSMS struct {
    client   *africastalking.SMS
    senderID string
}

func (s *africastalkingSMS) SendOTP(phone, code string) error {
    message := fmt.Sprintf("Your verification code is %s. Valid for 10 minutes.", code)
    _, err := s.client.Send(message, []string{phone}, s.senderID)
    return err
}

// Future: Twilio implementation
type twilioSMS struct {
    client *twilio.Client
}

func (s *twilioSMS) SendOTP(phone, code string) error {
    // Different API, same interface
}
```

**Decoupling Strategy:**
- All external services have interfaces
- Easy to add new providers (e.g., add Twilio alongside Africa's Talking)
- Easy to mock for tests

---

## Data Flow Examples

### Example 1: User Authentication (OTP Flow)

```
1. POST /auth/send-otp {"phone": "+233201234567"}
   ↓
2. AuthHandler.SendOTP()
   ├─ Validates E.164 format
   └─ Calls AuthService.SendOTP()
      ↓
3. AuthService.SendOTP()
   ├─ Checks rate limit (Redis: INCR otp:rate:{phone})
   ├─ Generates 6-digit code
   ├─ Stores in Redis (SET otp:{phone} {code} EX 600)
   └─ Calls SMSService.SendOTP()
      ↓
4. SMSService.SendOTP()
   └─ Sends SMS via Africa's Talking API
      ↓
5. Return 200 {"message": "OTP sent"}

---

Later: POST /auth/verify-otp {"phone": "+233...", "code": "123456"}
   ↓
6. AuthHandler.VerifyOTP()
   └─ Calls AuthService.VerifyOTP()
      ↓
7. AuthService.VerifyOTP()
   ├─ Gets code from Redis (GET otp:{phone})
   ├─ Compares with provided code
   ├─ Deletes from Redis (DEL otp:{phone})
   ├─ Finds or creates User (UserRepository)
   └─ Issues JWT (JWTService)
      ↓
8. Return 200 {"accessToken": "eyJ..."}
```

### Example 2: Load Creation & Matching

```
1. POST /loads {...} (authenticated)
   ↓
2. LoadHandler.CreateLoad()
   └─ Calls LoadService.CreateLoad()
      ↓
3. LoadService.CreateLoad()
   ├─ Calculates price (PricingService)
   ├─ Creates Load record (LoadRepository)
   ├─ Creates Stop records (StopRepository)
   └─ Enqueues matching job (asynq)
      ↓
4. Return 201 {load data}

---

Background (asynq worker):
5. MatchingJob runs
   └─ Calls MatchingService.FindBestMatch()
      ↓
6. MatchingService.FindBestMatch()
   ├─ Gets first pickup location
   ├─ Runs PostGIS query (TruckRepository)
   ├─ Scores trucks by distance + rating
   └─ Creates Trip record (TripRepository)
      ↓
7. Send push notification to driver (FCM)
```

---

## Database Schema (Current Phase 0, Evolving)

### Phase 0: Infrastructure Only
- No domain models yet
- PostGIS extension enabled
- Connection pooling configured

### Phase 1+: Core Models (Preview)
```sql
-- Users (all roles)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    role VARCHAR(50) NOT NULL,  -- shipper|driver|fleet_admin|owner_operator|admin
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(255),
    language VARCHAR(10) DEFAULT 'en',
    kyc_status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fleet companies
CREATE TABLE fleet_companies (
    id UUID PRIMARY KEY,
    business_name VARCHAR(255) NOT NULL,
    registration_doc TEXT,  -- S3 URL
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trucks (owned by fleet OR independent operator)
CREATE TABLE trucks (
    id UUID PRIMARY KEY,
    fleet_company_id UUID REFERENCES fleet_companies(id),  -- Nullable
    owner_user_id UUID REFERENCES users(id),               -- Nullable
    type VARCHAR(50) NOT NULL,  -- flatbed|box|tanker|reefer|tipper
    capacity_tons DECIMAL(10,2) NOT NULL,
    plate_number VARCHAR(50) UNIQUE NOT NULL,
    current_location GEOGRAPHY(POINT, 4326),  -- PostGIS
    status VARCHAR(50) DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraint: exactly one of fleet_company_id or owner_user_id must be set
    CHECK ((fleet_company_id IS NULL) <> (owner_user_id IS NULL))
);

-- Spatial index for fast nearest-truck queries
CREATE INDEX idx_trucks_location ON trucks USING GIST (current_location);
```

---

## Technology Decisions & Trade-offs

### Why Gin over other Go frameworks?
**Decision:** Use Gin for HTTP routing

**Pros:**
- Fastest Go HTTP framework (benchmarks: 40K req/s)
- Middleware ecosystem
- Built-in validation (validator/v10)
- Familiar API for devs coming from Express

**Cons:**
- Less "magical" than frameworks like Echo or Fiber
- Manual dependency injection (but we want explicit control)

**Rationale:** Performance matters for GPS tracking (1000s concurrent connections) and load matching (10ms response time goal).

---

### Why GORM over raw SQL?
**Decision:** Use GORM for simple CRUD, raw SQL for complex queries

**Pros:**
- Faster development for CRUD operations
- Automatic migrations (optional)
- Relationships handled well

**Cons:**
- Less control over queries
- Harder to optimize complex joins

**Rationale:** 80/20 rule — GORM for 80% of queries, raw SQL for the 20% that need PostGIS or performance tuning.

---

### Why asynq over other job queues?
**Decision:** Use asynq for background jobs (Phase 4+)

**Pros:**
- Redis-backed (we already have Redis)
- Built-in retry logic
- Scheduled/cron jobs
- Simple Go API

**Cons:**
- Less mature than RabbitMQ/Kafka
- Single point of failure (Redis)

**Rationale:** Simplicity > enterprise features for MVP. We can migrate to Kafka later if we need distributed guarantees.

---

## Scaling Strategy (Future Phases)

### Vertical Scaling (Phase 1-6)
- Single server handles 1000+ req/s
- Managed PostgreSQL (Neon) scales automatically
- Redis (Upstash) handles pub/sub for realtime

### Horizontal Scaling (Phase 10+)
```
          Load Balancer (AWS ALB)
                 /  |  \
               /    |    \
          Server1 Server2 Server3
            \      |      /
              \    |    /
            PostgreSQL (read replicas)
            Redis (cluster mode)
```

**Stateless servers:**
- JWT auth (no session storage)
- WebSocket connections can reconnect to any server (Redis pub/sub)

**Database read replicas:**
- Write to primary (trips, payments)
- Read from replicas (shipper viewing history)

---

## Security Considerations

### Authentication & Authorization
- **JWT tokens** with short expiry (7 days, refresh on activity)
- **RBAC middleware** checks roles before handler execution
- **Rate limiting** on OTP sends (3 per 10 min)

### Data Protection
- **Passwords:** N/A (phone-based auth only)
- **API keys:** Stored in env vars, never committed
- **KYC documents:** Presigned URLs with expiry, HTTPS only

### Network Security
- **HTTPS only** in production (Let's Encrypt)
- **CORS** configured for mobile app origins only
- **SQL injection:** Parameterized queries (GORM default)

---

## Monitoring & Observability (Phase 10+)

### Metrics
- **Request latency** (p50, p95, p99)
- **Error rate** (5xx responses)
- **Database query time**
- **Redis hit rate**

### Logging
- **Structured logs** (logrus/zap)
- **Log levels:** DEBUG (dev), INFO (staging), WARN/ERROR (prod)
- **No PII in logs** (redact phone numbers, payment info)

### Alerting
- **Sentry** for error tracking
- **Uptime monitoring** (UptimeRobot)
- **Database slow queries** (CloudWatch)

---

**Last Updated:** 2026-08-11 (Phase 0)
**Next Update:** Phase 1 (add auth flow diagrams)
