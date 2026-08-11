# Cross-Border Trucking Logistics Platform - Go Backend

Migrating from NestJS to Go/Gin for 10x performance improvement in Mali ↔ Ghana freight marketplace.

## Documentation

### For Developers
- **[docs/DEVELOPER-GUIDE.md](docs/DEVELOPER-GUIDE.md)** — ⭐ **Start here!** Comprehensive guide for new team members
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — System design, data flow, technology decisions
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — Code standards, testing, PR process
- **[CHANGELOG.md](CHANGELOG.md)** — Version history and release notes

### Planning & Migration
- **[IMPLEMENTATION-PLAN.md](IMPLEMENTATION-PLAN.md)** — Phases 0-5 (Foundation through GPS Tracking)
- **[IMPLEMENTATION-PLAN-PART2.md](IMPLEMENTATION-PLAN-PART2.md)** — Phases 6-12 (Payments through Launch)
- **[GO-MIGRATION-GUIDE.md](GO-MIGRATION-GUIDE.md)** — Technical migration guide from NestJS to Go
- **[trucking-logistics-app-guide.md](trucking-logistics-app-guide.md)** — Full product requirements and domain understanding

## Quick Start

```bash
# Install dependencies
go mod tidy

# Set up environment
cp .env.example .env
# Edit .env with your credentials

# Run connection health check
go run cmd/check/main.go

# Run migrations
make migrate-up

# Start server
go run cmd/server/main.go
```

## Project Status

**Phase 0: Foundation & Setup** - ✅ **COMPLETE with Enhanced Documentation** (2026-08-11)

Current structure includes:
- ✅ Proper Go project layout (`cmd/`, `pkg/`, `internal/`)
- ✅ Configuration management with environment variables
- ✅ PostgreSQL + PostGIS database setup
- ✅ Redis cache client with pub/sub
- ✅ HTTP server with Gin framework
- ✅ Health check CLI tool
- ✅ Docker Compose for local development
- ✅ Response helpers and utilities
- ✅ Graceful shutdown handling
- ✅ **NEW: Comprehensive team documentation (18,500+ words)**
  - Developer onboarding guide
  - System architecture overview
  - Daily development quick reference
  - Contribution guidelines
  - Living documentation system

**What's Special:** This project is now **team-ready**. Any developer can join and be productive within 1-2 days thanks to comprehensive, living documentation that evolves with the codebase.

**Next:** Phase 1 - Authentication & User Management (OTP, JWT, RBAC)

## Timeline

**Target:** 12 weeks from start to pilot launch
- Weeks 1-2: Foundation + Auth
- Weeks 3-5: Core domain + Matching
- Weeks 6-7: GPS Tracking + Payments
- Weeks 8-10: Admin + Testing
- Weeks 11-12: Deployment + Pilot

## Key Technical Decisions

| Area | Choice | Reason |
|---|---|---|
| Database | PostgreSQL + PostGIS | Geospatial queries for truck matching |
| Cache | Redis | OTP storage, live location cache, pub/sub |
| Queue | asynq | Background jobs (matching, notifications) |
| Payments | Paystack + Flutterwave | Mobile money support, cross-border |
| Storage | Cloudflare R2 | KYC docs, customs documents |
| SMS | Africa's Talking | Better African coverage + pricing |

## Architecture

```
Mobile Apps (RN) → API Gateway (Gin) → Services → PostgreSQL/Redis
                  ↓
              WebSocket → GPS Tracking → Shipper App
```

## Critical Path Items

1. **Start immediately:** Flutterwave cross-border KYC approval (2-4 weeks)
2. **Week 5:** PostGIS query performance testing
3. **Week 6:** Field test GPS tracking in low-connectivity areas
4. **Week 7:** Payment provider sandbox testing

## Contact

For questions about the implementation plan, see the detailed guides in the docs folder.
