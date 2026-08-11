# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned
- Phase 1: Authentication & User Management
- Phase 2: Core Domain Models (Fleet, Trucks, Drivers)
- Phase 3: Load & Trip System
- Phase 4: Matching Engine
- Phase 5: Live GPS Tracking

---

## [0.1.1] - 2026-08-11

### Added - JWT Authentication & R2 Storage

#### Authentication
- JWT token generation and verification (`pkg/auth/jwt.go`)
- JWT Manager with HMAC-SHA256 signing
- Claims structure: User ID, Email, Role, Standard JWT claims
- Token refresh functionality
- Cryptographic key generator (`cmd/keygen/main.go`)
- JWT configuration validation (minimum 32 characters)
- JWT testing in health check CLI

#### Cloud Storage
- **Cloudflare R2 integration** with S3-compatible API (`pkg/storage/r2.go`)
- R2 client using AWS SDK v2
- Operations: Upload, GetPresignedURL, Delete, List, TestConnection
- R2 bucket accessibility check in health check CLI
- Configurable public URL for file access

#### Documentation
- JWT Setup Guide (`docs/JWT-SETUP.md`)
- Security best practices for JWT
- Token generation and verification examples
- Cloudflare R2 Setup Guide (`docs/R2-SETUP.md`)
- R2 folder structure recommendations
- Cost analysis and security practices

#### Dependencies
- `github.com/golang-jwt/jwt/v5` v5.3.1 - JWT implementation
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for R2 (S3-compatible)
- `github.com/aws/aws-sdk-go-v2/service/s3` - S3 service client
- `github.com/aws/aws-sdk-go-v2/credentials` - Static credentials provider

### Changed
- Health check CLI now tests JWT functionality
- Health check CLI includes optional R2 testing
- `.env.example` updated with JWT key generation instructions
- Configuration validation includes JWT secret length check

### Security
- Cryptographically secure JWT secret generation
- JWT secret must be minimum 32 characters
- Token expiry configuration (default: 7 days)
- Secure R2 credential handling

---

## [0.1.0] - 2026-08-11

### Added - Phase 0: Foundation & Setup ✅

#### Project Infrastructure
- Initial Go project structure (`cmd/`, `pkg/`, `internal/`)
- Environment configuration management with validation
- Docker Compose setup for PostgreSQL + Redis
- Makefile with common development commands
- Comprehensive `.gitignore` for Go projects

#### Database & Cache
- PostgreSQL connection with GORM ORM
- **Neon PostgreSQL integration** with serverless architecture
- PostGIS extension auto-enablement for geospatial queries
- Connection pooling configuration optimized for Neon
- **Upstash Redis integration** with serverless architecture
- Redis client with pub/sub support (compatible with Upstash)
- Context-aware cache operations (Get, Set, Del, Incr, Expire)
- Support for both native Redis protocol and REST API

#### HTTP Server
- Gin-based HTTP server with graceful shutdown
- Health check endpoint: `GET /health`
- API v1 routing group structure
- Standard JSON response helpers (Success, Error, BadRequest, etc.)
- CORS and logging middleware

#### Developer Tools
- Connection health check CLI (`cmd/check/main.go`)
- PostGIS spatial query verification
- Redis command testing
- Development server entry point (`cmd/server/main.go`)

#### Documentation
- Comprehensive Developer Guide (`docs/DEVELOPER-GUIDE.md`)
- Architecture Overview (`docs/ARCHITECTURE.md`)
- Quick Reference (`docs/QUICK-REFERENCE.md`)
- Documentation Index (`docs/DOCUMENTATION-INDEX.md`)
- Contributing Guidelines (`CONTRIBUTING.md`)
- **Neon PostgreSQL Setup Guide** (`docs/NEON-SETUP.md`)
- **Neon Benefits & Decision Rationale** (`docs/NEON-BENEFITS.md`)
- **Upstash Redis Setup Guide** (`docs/UPSTASH-SETUP.md`)
- Implementation Plan (12-week roadmap)
- Migration Guide from NestJS
- Changelog template
- README with quick start instructions
- Phase 0 completion summary

#### Dependencies
- `github.com/gin-gonic/gin` v1.12.0 - HTTP framework
- `gorm.io/gorm` v1.31.2 - ORM
- `gorm.io/driver/postgres` v1.6.2 - PostgreSQL driver
- `github.com/redis/go-redis/v9` v9.22.0 - Redis client
- `github.com/joho/godotenv` v1.5.1 - Environment variable loader

### Technical Details

**Configuration:**
- Struct-based config with environment variable loading
- Validation of required fields (DATABASE_URL, JWT_SECRET)
- Support for development and production environments

**Database:**
- Connection string format: `postgresql://user:pass@host:port/db?sslmode=disable`
- Automatic PostGIS extension creation
- Health check includes PostGIS version and spatial query test

**Redis:**
- URL-based connection configuration
- Context propagation for timeouts and cancellation
- Pub/sub capabilities prepared for realtime features

**Code Quality:**
- Interface-based design for modularity
- Dependency injection pattern
- Comprehensive inline documentation
- Error wrapping with context

### Development Setup

```bash
# Clone and setup
git clone <repo>
cd logistic-app-go
cp .env.example .env

# Start services
docker-compose up -d

# Verify connections
go run cmd/check/main.go

# Start server
go run cmd/server/main.go
```

---

## Template for Future Entries

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features

### Changed
- Changes to existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Removed features

### Fixed
- Bug fixes

### Security
- Security improvements
```

---

[Unreleased]: https://github.com/yourusername/logistic-app-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/yourusername/logistic-app-go/releases/tag/v0.1.0
