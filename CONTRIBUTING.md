# Contributing Guide
**Cross-Border Trucking Logistics Platform**

Thank you for contributing! This guide will help you get started and follow our development standards.

---

## Table of Contents
1. [Getting Started](#getting-started)
2. [Development Workflow](#development-workflow)
3. [Code Standards](#code-standards)
4. [Testing Requirements](#testing-requirements)
5. [Commit Guidelines](#commit-guidelines)
6. [Pull Request Process](#pull-request-process)
7. [Documentation](#documentation)

---

## Getting Started

### First-Time Contributors

1. **Read the documentation**
   - [README.md](README.md) - Project overview
   - [docs/DEVELOPER-GUIDE.md](docs/DEVELOPER-GUIDE.md) - Comprehensive developer guide
   - [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design

2. **Set up your environment**
   ```bash
   # Fork and clone the repository
   git clone https://github.com/YOUR_USERNAME/logistic-app-go.git
   cd logistic-app-go
   
   # Install dependencies
   go mod download
   
   # Set up environment
   cp .env.example .env
   # Edit .env with your local credentials
   
   # Start local services
   docker-compose up -d
   
   # Verify setup
   go run cmd/check/main.go
   ```

3. **Run the server**
   ```bash
   go run cmd/server/main.go
   # Visit http://localhost:8080/health
   ```

---

## Development Workflow

### Branch Naming Convention

```
type/scope-description

Types:
- feat/     New feature
- fix/      Bug fix
- refactor/ Code refactoring
- docs/     Documentation only
- test/     Adding tests
- chore/    Maintenance tasks

Examples:
- feat/auth-otp-verification
- fix/matching-algorithm-edge-case
- refactor/payment-service-interface
- docs/update-api-endpoints
```

### Creating a New Feature

1. **Create a branch**
   ```bash
   git checkout -b feat/driver-rating-system
   ```

2. **Follow the layered architecture**
   ```
   internal/rating/
   ├── dto.go           # Request/response structs
   ├── model.go         # Domain model
   ├── repository.go    # Database interface + implementation
   ├── service.go       # Business logic interface + implementation
   └── handler.go       # HTTP handlers
   ```

3. **Write interfaces first**
   ```go
   // Define the contract
   type RatingService interface {
       CreateRating(ctx context.Context, req CreateRatingRequest) error
       GetDriverRating(ctx context.Context, driverID uuid.UUID) (float64, error)
   }
   ```

4. **Implement with dependency injection**
   ```go
   type ratingService struct {
       ratingRepo RatingRepository
       driverRepo DriverRepository
   }
   
   func NewRatingService(
       ratingRepo RatingRepository,
       driverRepo DriverRepository,
   ) RatingService {
       return &ratingService{
           ratingRepo: ratingRepo,
           driverRepo: driverRepo,
       }
   }
   ```

5. **Wire up in main.go**
   ```go
   // cmd/server/main.go
   ratingRepo := rating.NewRepository(db)
   ratingService := rating.NewService(ratingRepo, driverRepo)
   ratingHandler := rating.NewHandler(ratingService)
   
   v1.POST("/ratings", authMiddleware, ratingHandler.CreateRating)
   ```

---

## Code Standards

### Go Style Guidelines

**Follow official Go conventions:**
- Run `gofmt` before committing
- Use `golangci-lint` for linting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)

**Naming:**
```go
// ✅ Good
type UserRepository interface { ... }
type userRepository struct { ... }
func NewUserRepository() UserRepository { ... }

// ❌ Bad
type User_Repository interface { ... }
type Userrepository struct { ... }
func new_user_repository() UserRepository { ... }
```

**Error Handling:**
```go
// ✅ Good - Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// ❌ Bad - Losing error context
if err != nil {
    return errors.New("error occurred")
}
```

**Comments:**
```go
// ✅ Good - Doc comments for exported items
// CreateUser creates a new user account with the provided phone number.
// It returns an error if the phone number is already registered.
func CreateUser(ctx context.Context, phone string) (*User, error) {
    // Implementation...
}

// ❌ Bad - No doc comment
func CreateUser(ctx context.Context, phone string) (*User, error) {
    // Implementation...
}
```

**Context Propagation:**
```go
// ✅ Good - Always accept context as first parameter
func (s *service) DoSomething(ctx context.Context, id uuid.UUID) error {
    result, err := s.repo.Find(ctx, id)
    // ...
}

// ❌ Bad - Missing context
func (s *service) DoSomething(id uuid.UUID) error {
    result, err := s.repo.Find(id)
    // ...
}
```

### Modularity Checklist

Before submitting a PR, verify:

- [ ] All external dependencies use interfaces
- [ ] No global variables (except config, logger)
- [ ] Services receive dependencies via constructor
- [ ] No direct database access in handlers
- [ ] No business logic in repositories
- [ ] Implementations can be swapped via interfaces

**Example of Good Modularity:**
```go
// ✅ Service depends on interface, not concrete type
type PaymentService interface {
    ChargeShipper(ctx context.Context, loadID uuid.UUID, amount float64) error
}

type loadService struct {
    paymentService PaymentService  // Interface, not *paystackPayment
}

// Easy to swap Paystack → Flutterwave
// Easy to mock in tests
```

---

## Testing Requirements

### Test Coverage Goals
- **Unit tests:** 70%+ coverage for business logic
- **Integration tests:** All HTTP endpoints
- **No tests required for:** Simple CRUD repositories (covered by integration tests)

### Writing Unit Tests

**Location:** Same package as implementation (`*_test.go`)

```go
// internal/auth/service_test.go
package auth

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock the dependency
type MockOTPService struct {
    mock.Mock
}

func (m *MockOTPService) Generate() string {
    args := m.Called()
    return args.String(0)
}

// Test the business logic
func TestVerifyOTP_Success(t *testing.T) {
    // Arrange
    mockCache := &MockCache{}
    mockCache.On("Get", mock.Anything, "otp:+233201234567").Return("123456", nil)
    
    service := NewOTPService(mockCache)
    
    // Act
    err := service.VerifyOTP(context.Background(), "+233201234567", "123456")
    
    // Assert
    assert.NoError(t, err)
    mockCache.AssertExpectations(t)
}

func TestVerifyOTP_InvalidCode(t *testing.T) {
    // Test failure case
}
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/auth

# With coverage
go test -cover ./...

# Verbose
go test -v ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Naming Convention
```go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T) { }

Examples:
- TestVerifyOTP_ValidCode_ReturnsNoError
- TestCreateLoad_InvalidWeight_ReturnsError
- TestMatchTruck_NoAvailableTrucks_ReturnsNil
```

---

## Commit Guidelines

### Commit Message Format

```
type(scope): subject

body (optional)

footer (optional)
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `docs`: Documentation changes
- `test`: Adding/updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples:**
```
feat(auth): implement OTP verification with Redis

- Add OTP generation and storage
- Implement rate limiting (3 attempts per 10 min)
- Add SMS sending via Africa's Talking
- Include unit tests for verification logic

Closes #42
```

```
fix(matching): handle edge case where no trucks available

Previously the system would crash if no trucks were within range.
Now returns a clear error message to the shipper.

Fixes #87
```

```
refactor(payment): extract payment provider interface

This allows us to easily swap between Paystack and Flutterwave
based on transaction currency and location.
```

### Commit Best Practices

- ✅ **One logical change per commit**
- ✅ **Present tense:** "Add feature" not "Added feature"
- ✅ **Keep subject under 72 characters**
- ✅ **Reference issue numbers**
- ❌ **Don't commit commented-out code**
- ❌ **Don't mix refactoring with feature changes**

---

## Pull Request Process

### Before Creating a PR

1. **Run tests and linting**
   ```bash
   go test ./...
   golangci-lint run
   go mod tidy
   ```

2. **Update documentation**
   - Add/update API docs if endpoints changed
   - Update CHANGELOG.md
   - Update DEVELOPER-GUIDE.md if architecture changed

3. **Self-review your code**
   - Remove debug statements
   - Check for hardcoded values
   - Verify error messages are clear

### PR Template

```markdown
## Description
Brief description of what this PR does.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## How to Test
1. Step-by-step instructions
2. Expected behavior
3. Edge cases to verify

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] No breaking changes (or documented)
- [ ] Follows code style guidelines

## Related Issues
Closes #123
Related to #456
```

### Review Process

1. **Automated checks must pass**
   - Tests
   - Linting
   - Build

2. **At least 1 approval required**
   - Reviewer checks:
     - Code quality
     - Test coverage
     - Documentation
     - Security concerns

3. **Squash merge to main**
   - Keep main branch history clean

---

## Documentation

### What to Document

**Always document:**
- Public functions and types (godoc format)
- Complex algorithms (inline comments)
- Non-obvious decisions (why, not what)
- API endpoints (update API.md)
- Database schema changes (migrations + ARCHITECTURE.md)

**Example godoc:**
```go
// FindNearestTrucks returns up to `limit` available trucks within the specified
// radius of the given location, sorted by distance. It uses PostGIS ST_Distance
// for geospatial calculations.
//
// The location should be provided as a WGS84 coordinate pair (longitude, latitude).
// Radius is specified in meters.
//
// Returns an empty slice if no trucks are found within the radius.
func FindNearestTrucks(ctx context.Context, location Point, radiusMeters float64, limit int) ([]Truck, error) {
    // ...
}
```

### Updating Living Documentation

When your PR affects these areas, update the corresponding docs:

| Change | Document to Update |
|---|---|
| New API endpoint | `docs/API.md` |
| Database schema | `docs/ARCHITECTURE.md` + migration file |
| New domain concept | `docs/DEVELOPER-GUIDE.md` |
| External service | `docs/ARCHITECTURE.md` (External Services) |
| Configuration var | `.env.example` + `README.md` |

---

## Questions?

- **Slack:** #engineering (if using Slack)
- **Email:** engineering@yourcompany.com
- **Issues:** Open a GitHub issue with `question` label

---

**Thank you for contributing!** 🚀
