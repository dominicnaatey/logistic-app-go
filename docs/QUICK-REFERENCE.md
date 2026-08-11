# Quick Reference
**Handy commands and patterns for daily development**

---

## Common Commands

### Development
```bash
# Start services
docker-compose up -d

# Check connections
go run cmd/check/main.go
make check

# Run server
go run cmd/server/main.go
make dev

# Run tests
go test ./...
make test

# Build binaries
go build -o bin/server cmd/server/main.go
make build
```

### Dependencies
```bash
# Add a new package
go get github.com/package/name

# Update dependencies
go get -u ./...

# Tidy and verify
go mod tidy
go mod verify
```

### Docker
```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Restart a single service
docker-compose restart postgres
```

### Database
```bash
# Connect to PostgreSQL
docker exec -it logistic_postgres psql -U logistic_user -d logistic_app

# Run migrations (Phase 1+)
migrate -path migrations -database "postgresql://..." up

# Rollback last migration
migrate -path migrations -database "postgresql://..." down 1
```

### Redis
```bash
# Connect to Redis
docker exec -it logistic_redis redis-cli

# Common commands
PING
GET otp:+233201234567
SET test_key "value" EX 60
DEL test_key
```

---

## Project Structure Patterns

### Adding a New Domain Feature

1. **Create the package structure**
   ```
   internal/rating/
   ├── dto.go          # Request/response types
   ├── model.go        # Domain model
   ├── repository.go   # Data access interface
   ├── service.go      # Business logic interface
   └── handler.go      # HTTP handlers
   ```

2. **Define DTOs** (`dto.go`)
   ```go
   package rating
   
   type CreateRatingRequest struct {
       TripID   uuid.UUID `json:"trip_id" binding:"required"`
       Score    int       `json:"score" binding:"required,min=1,max=5"`
       Comment  string    `json:"comment"`
   }
   
   type RatingResponse struct {
       ID        uuid.UUID `json:"id"`
       Score     int       `json:"score"`
       Comment   string    `json:"comment"`
       CreatedAt time.Time `json:"created_at"`
   }
   ```

3. **Define Model** (`model.go`)
   ```go
   package rating
   
   type Rating struct {
       ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
       TripID    uuid.UUID `gorm:"not null;unique"`
       RaterID   uuid.UUID `gorm:"not null"`
       RateeID   uuid.UUID `gorm:"not null"`
       Score     int       `gorm:"not null"`
       Comment   string
       CreatedAt time.Time
   }
   ```

4. **Define Repository Interface** (`repository.go`)
   ```go
   package rating
   
   type Repository interface {
       Create(ctx context.Context, rating *Rating) error
       FindByDriverID(ctx context.Context, driverID uuid.UUID) ([]Rating, error)
   }
   
   type repository struct {
       db *gorm.DB
   }
   
   func NewRepository(db *gorm.DB) Repository {
       return &repository{db: db}
   }
   
   func (r *repository) Create(ctx context.Context, rating *Rating) error {
       return r.db.WithContext(ctx).Create(rating).Error
   }
   ```

5. **Define Service Interface** (`service.go`)
   ```go
   package rating
   
   type Service interface {
       CreateRating(ctx context.Context, req CreateRatingRequest) error
       GetDriverRating(ctx context.Context, driverID uuid.UUID) (float64, error)
   }
   
   type service struct {
       repo Repository
   }
   
   func NewService(repo Repository) Service {
       return &service{repo: repo}
   }
   
   func (s *service) CreateRating(ctx context.Context, req CreateRatingRequest) error {
       rating := &Rating{
           TripID:  req.TripID,
           Score:   req.Score,
           Comment: req.Comment,
       }
       return s.repo.Create(ctx, rating)
   }
   ```

6. **Add Handlers** (`handler.go`)
   ```go
   package rating
   
   type Handler struct {
       service Service
   }
   
   func NewHandler(service Service) *Handler {
       return &Handler{service: service}
   }
   
   func (h *Handler) CreateRating(c *gin.Context) {
       var req CreateRatingRequest
       if err := c.ShouldBindJSON(&req); err != nil {
           response.BadRequest(c, "Invalid request")
           return
       }
       
       if err := h.service.CreateRating(c.Request.Context(), req); err != nil {
           response.InternalError(c, "Failed to create rating")
           return
       }
       
       response.Created(c, gin.H{"message": "Rating created"})
   }
   ```

7. **Wire up in main** (`cmd/server/main.go`)
   ```go
   // Repositories
   ratingRepo := rating.NewRepository(db)
   
   // Services
   ratingService := rating.NewService(ratingRepo)
   
   // Handlers
   ratingHandler := rating.NewHandler(ratingService)
   
   // Routes
   v1.POST("/ratings", authMiddleware, ratingHandler.CreateRating)
   v1.GET("/drivers/:id/rating", ratingHandler.GetDriverRating)
   ```

---

## Common Patterns

### Context Timeouts
```go
// Always set timeouts for external calls
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := externalService.Call(ctx, params)
```

### Error Wrapping
```go
// Wrap errors with context for debugging
if err != nil {
    return fmt.Errorf("failed to fetch user %s: %w", userID, err)
}
```

### Validation
```go
// Use Gin's built-in validation
type Request struct {
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age" binding:"required,min=18,max=120"`
}

if err := c.ShouldBindJSON(&req); err != nil {
    response.ValidationError(c, err.Error())
    return
}
```

### Database Transactions
```go
// Use transactions for multi-step operations
err := db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    
    if err := tx.Create(&profile).Error; err != nil {
        return err
    }
    
    return nil
})
```

### Redis Caching
```go
// Cache with expiration
key := fmt.Sprintf("user:%s", userID)
err := cache.Set(ctx, key, userData, 1*time.Hour)

// Get from cache
data, err := cache.Get(ctx, key)
if err != nil {
    // Cache miss, fetch from database
}
```

---

## Testing Patterns

### Unit Test with Mocks
```go
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func TestGetUser(t *testing.T) {
    mockRepo := new(MockRepository)
    mockRepo.On("FindByID", mock.Anything, testUserID).Return(&testUser, nil)
    
    service := NewService(mockRepo)
    user, err := service.GetUser(context.Background(), testUserID)
    
    assert.NoError(t, err)
    assert.Equal(t, testUser.Name, user.Name)
    mockRepo.AssertExpectations(t)
}
```

### HTTP Handler Test
```go
func TestCreateRating(t *testing.T) {
    // Setup
    router := gin.New()
    handler := NewHandler(mockService)
    router.POST("/ratings", handler.CreateRating)
    
    // Request
    body := `{"trip_id":"...","score":5}`
    req := httptest.NewRequest("POST", "/ratings", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Execute
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, 201, w.Code)
}
```

---

## Environment Variables Reference

```bash
# Server
GO_ENV=development          # development | production
PORT=8080                   # HTTP port

# Database (Neon PostgreSQL)
DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=require

# Redis (Upstash)
REDIS_URL=redis://default:token@endpoint.upstash.io:6379
REDIS_TOKEN=your-token      # Upstash calls it "token"

# JWT
JWT_SECRET=your-secret-key-min-32-chars
JWT_EXPIRES_IN=168h        # 7 days

# SMS (Africa's Talking)
AT_USERNAME=sandbox
AT_API_KEY=your-api-key
AT_SENDER_ID=               # Optional

# Storage (Cloudflare R2)
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET_NAME=
R2_PUBLIC_URL=

# Payments
PAYSTACK_SECRET_KEY=
PAYSTACK_PUBLIC_KEY=
FLUTTERWAVE_SECRET_KEY=
FLUTTERWAVE_PUBLIC_KEY=
FLUTTERWAVE_ENCRYPTION_KEY=

# Monitoring
SENTRY_DSN=                 # Optional
```

---

## Git Workflow

### Feature Development
```bash
# Create feature branch
git checkout -b feat/driver-ratings

# Make changes, commit often
git add .
git commit -m "feat(rating): implement rating creation"

# Push to remote
git push origin feat/driver-ratings

# Create PR on GitHub
```

### Keeping Branch Updated
```bash
# Fetch latest changes
git fetch origin

# Rebase on main
git rebase origin/main

# Force push (if already pushed)
git push --force-with-lease
```

---

## Debugging Tips

### View Logs
```bash
# Docker logs
docker-compose logs -f

# Specific service
docker-compose logs -f postgres
```

### Database Debugging
```sql
-- Show current connections
SELECT * FROM pg_stat_activity;

-- Show slow queries
SELECT * FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;

-- Check PostGIS version
SELECT PostGIS_Version();
```

### Redis Debugging
```bash
# Monitor all commands
docker exec -it logistic_redis redis-cli MONITOR

# Check memory usage
docker exec -it logistic_redis redis-cli INFO memory

# List all keys
docker exec -it logistic_redis redis-cli KEYS '*'
```

---

## Useful Links

- [Gin Documentation](https://gin-gonic.com/docs/)
- [GORM Documentation](https://gorm.io/docs/)
- [PostGIS Documentation](https://postgis.net/documentation/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go.html)

---

**Last Updated:** 2026-08-11
