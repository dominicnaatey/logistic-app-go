# Neon PostgreSQL Setup Guide
**Serverless PostgreSQL with PostGIS for Cross-Border Logistics**

Last Updated: 2026-08-11

---

## Why Neon?

**Neon is perfect for this project because:**
- ✅ **PostGIS pre-enabled** — No manual extension setup needed
- ✅ **Serverless** — Auto-scales to zero when not in use, saves costs
- ✅ **Database branching** — Create test databases instantly (like Git branches)
- ✅ **Connection pooling** — Built-in pooler handles high concurrency
- ✅ **Geographic replication** — Deploy close to your users (Ghana/Mali)
- ✅ **Generous free tier** — 3 GB storage, 300 compute hours/month

---

## Getting Started with Neon

### 1. Create a Neon Account

1. Go to **https://console.neon.tech**
2. Sign up (free tier available)
3. Verify your email

### 2. Create Your Project

1. Click **"New Project"**
2. Configure:
   - **Project Name:** `logistic-app-production` (or your preferred name)
   - **Region:** Select closest to your users
     - For Ghana: `US East (Ohio)` or `EU (Frankfurt)`
     - For West Africa optimization: `EU (Frankfurt)` is often best
   - **PostgreSQL version:** 15 (recommended)
   - **Compute size:** Start with `Shared` (free tier)

3. Click **"Create Project"**

### 3. Enable PostGIS

Neon has PostGIS pre-installed, but you need to enable it:

**Option A: Via Neon Console (SQL Editor)**
```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

**Option B: Via our health check (automatic)**
Our `pkg/database/postgres.go` enables PostGIS automatically on connection:
```go
// Runs on first connection
db.Exec("CREATE EXTENSION IF NOT EXISTS postgis;")
```

**Verify PostGIS is enabled:**
```sql
SELECT PostGIS_Version();
-- Should return: 3.3 USE_GEOS=1 USE_PROJ=1 USE_STATS=1
```

### 4. Get Your Connection String

1. In Neon console, go to your project
2. Click **"Connection Details"**
3. Copy the **connection string**:

```
postgresql://username:password@ep-random-string-pooler.us-east-2.aws.neon.tech/neondb?sslmode=require
```

**Important Notes:**
- Use the **pooled connection** (`-pooler.` in hostname) for production
- Use the **direct connection** (no `-pooler`) for migrations
- Always use `sslmode=require` for Neon (it's required)

### 5. Configure Your App

1. **Copy the connection string**
   ```bash
   cp .env.example .env
   ```

2. **Update `.env`**
   ```env
   DATABASE_URL=postgresql://your-username:your-password@ep-xxx-pooler.us-east-2.aws.neon.tech/neondb?sslmode=require
   ```

3. **Test the connection**
   ```bash
   go run cmd/check/main.go
   ```

   You should see:
   ```
   ✓ PostgreSQL connected
   ✓ PostGIS version: 3.3 USE_GEOS=1 USE_PROJ=1 USE_STATS=1
   ✓ Spatial queries working
   ```

---

## Connection Strings Explained

### Pooled Connection (Recommended for App)
```
postgresql://user:pass@ep-xxx-pooler.us-east-2.aws.neon.tech/neondb?sslmode=require
                         ^^^^^^^^
```
- Use for your Go application
- Handles connection pooling automatically
- Better performance with many concurrent requests
- Default pool size: 100 connections

### Direct Connection (For Migrations)
```
postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require
                         (no -pooler)
```
- Use for database migrations
- Direct connection to database
- Required for DDL operations (CREATE TABLE, etc.)

---

## Database Branching (Neon's Killer Feature)

### What is Branching?

Neon lets you create instant database copies (branches), like Git branches for your database.

**Use cases:**
- **Development:** Each developer gets their own branch
- **Testing:** Test migrations before applying to production
- **Feature branches:** Isolated data for each feature
- **CI/CD:** Automated testing with fresh data

### Creating a Branch

**Via Neon Console:**
1. Go to your project
2. Click **"Branches"**
3. Click **"Create Branch"**
4. Name it (e.g., `dev-dominic`, `feat-matching-engine`)
5. Get the new connection string

**Via Neon CLI:**
```bash
# Install Neon CLI
npm install -g neonctl

# Create a branch
neonctl branches create --name dev-dominic --project-id <your-project-id>

# Get connection string
neonctl connection-string dev-dominic
```

### Using Branches in Development

**In `.env`:**
```env
# Production
DATABASE_URL=postgresql://...@ep-xxx-pooler.../neondb?sslmode=require

# Development branch
# DATABASE_URL=postgresql://...@ep-yyy-pooler.../neondb?sslmode=require
```

**Best Practice:**
- `main` branch → production data
- `dev` branch → shared development data
- Personal branches → individual developer data

---

## Neon Configuration Tips

### 1. Connection Pooling Settings

Our GORM setup works great with Neon's pooler:

```go
// pkg/database/postgres.go
sqlDB.SetMaxIdleConns(10)   // Keep 10 idle connections
sqlDB.SetMaxOpenConns(100)  // Max 100 open connections
```

**Neon pooler handles:**
- Connection queuing when pool is full
- Automatic connection recycling
- Load balancing across compute instances

### 2. Auto-Suspend & Warm-Up

Neon auto-suspends after 5 minutes of inactivity (free tier).

**First request after suspension:**
- Takes 1-3 seconds to wake up
- Subsequent requests are instant

**For production (paid tier):**
- Disable auto-suspend
- Or set longer suspend delay (e.g., 1 hour)

**Handle warm-up in code:**
```go
// Health check keeps database warm
go func() {
    ticker := time.NewTicker(4 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        db.Exec("SELECT 1")  // Ping to prevent suspend
    }
}()
```

### 3. Compute Size Recommendations

| Stage | Compute Size | Monthly Cost |
|---|---|---|
| Development | Shared (0.25 vCPU) | Free |
| Staging | Fixed: 0.5 vCPU | ~$20 |
| Production (MVP) | Fixed: 1 vCPU | ~$70 |
| Production (Scale) | Autoscaling: 1-4 vCPU | ~$100-300 |

**Upgrade when:**
- Response times > 500ms
- Concurrent users > 100
- GPS pings > 500/sec

---

## Migration Strategy with Neon

### Phase 1-9: Neon Main Branch

Use Neon's main branch for development and testing.

### Phase 10: Production Setup

1. **Create production branch**
   ```bash
   neonctl branches create --name production --parent main
   ```

2. **Configure deployment**
   ```env
   # Staging
   DATABASE_URL=postgresql://...@ep-staging-pooler.../neondb?sslmode=require
   
   # Production
   DATABASE_URL=postgresql://...@ep-prod-pooler.../neondb?sslmode=require
   ```

3. **Set up backups**
   - Neon takes automatic backups (7-day retention on free tier)
   - Upgrade to paid tier for longer retention (30+ days)

---

## Monitoring & Performance

### Via Neon Console

1. **Metrics Dashboard**
   - Active connections
   - Query latency (p50, p95, p99)
   - Database size
   - Compute usage

2. **Query Stats**
   ```sql
   SELECT * FROM pg_stat_statements 
   ORDER BY mean_exec_time DESC 
   LIMIT 10;
   ```

3. **Connection Stats**
   ```sql
   SELECT * FROM pg_stat_activity;
   ```

### Performance Tips

**Indexes for geospatial queries:**
```sql
-- Create spatial index on truck locations
CREATE INDEX idx_trucks_location ON trucks USING GIST (current_location);

-- Create compound index for common query
CREATE INDEX idx_trucks_status_location ON trucks (status) INCLUDE (current_location);
```

**Query optimization:**
```sql
-- Use EXPLAIN ANALYZE to understand query plans
EXPLAIN ANALYZE
SELECT id, ST_Distance(current_location, ST_MakePoint(-0.1870, 5.6037)::geography) AS dist
FROM trucks
WHERE status = 'available'
ORDER BY dist
LIMIT 10;
```

---

## Cost Management

### Free Tier Limits
- **Storage:** 3 GB
- **Compute hours:** 300/month (~10 hours/day)
- **Data transfer:** 5 GB/month
- **Branches:** 10

### Staying Within Free Tier

**Tips:**
1. Enable auto-suspend (5 min default)
2. Delete unused branches
3. Use connection pooling (reduces compute usage)
4. Archive old data (keep DB under 3 GB)

**When to upgrade:**
- Storage > 3 GB
- Consistent 24/7 usage
- Need longer backup retention
- Need more branches for team

---

## Troubleshooting

### "Database does not exist"

**Solution:**
```sql
CREATE DATABASE neondb;
```

Or use the default database name Neon created.

### "Connection pooler timeout"

**Causes:**
- Too many concurrent connections
- Long-running queries blocking pool

**Solutions:**
- Increase `MaxOpenConns` in Go
- Optimize slow queries
- Upgrade compute size

### "SSL connection required"

**Always use:**
```
?sslmode=require
```

Neon requires SSL for all connections.

### PostGIS functions not found

**Enable PostGIS:**
```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

Should be automatic via our `pkg/database/postgres.go`.

---

## Local Development Alternative

If you need offline development:

**Option 1: Use Neon Branch**
- Create a `dev` branch
- Use for all local development
- Never touches production data

**Option 2: Use Docker (Offline)**
```bash
docker-compose up -d
```

Update `.env`:
```env
DATABASE_URL=postgresql://logistic_user:logistic_pass@localhost:5432/logistic_app?sslmode=disable
```

**Trade-offs:**
- Docker: Works offline, but need to manually enable PostGIS
- Neon: Cloud-based, always up-to-date, PostGIS pre-enabled

---

## Useful Links

- **Neon Console:** https://console.neon.tech
- **Neon Documentation:** https://neon.tech/docs
- **PostGIS on Neon:** https://neon.tech/docs/extensions/postgis
- **Connection Pooling:** https://neon.tech/docs/connect/connection-pooling
- **Database Branching:** https://neon.tech/docs/introduction/branching

---

**Last Updated:** 2026-08-11 (Phase 0)
**Next Update:** Phase 1 (add migration setup)
