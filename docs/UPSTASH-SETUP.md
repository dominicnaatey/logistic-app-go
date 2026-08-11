# Upstash Redis Setup Guide
**Serverless Redis for Cross-Border Logistics**

Last Updated: 2026-08-11

---

## Why Upstash Redis?

**Upstash is perfect for this project because:**
- ✅ **Serverless** — Pay per request, auto-scales to zero
- ✅ **Global edge network** — Low latency from Ghana/Mali
- ✅ **REST API** — Easy integration, no connection management
- ✅ **Native Redis** — Also supports standard Redis protocol
- ✅ **Free tier** — 10,000 commands/day, perfect for development
- ✅ **Built-in TLS** — Secure by default

---

## Getting Started with Upstash

### 1. Create an Upstash Account

1. Go to **https://console.upstash.com**
2. Sign up (free tier available)
3. Verify your email

### 2. Create Your Redis Database

1. Click **"Create Database"**
2. Configure:
   - **Name:** `logistic-app-cache` (or your preferred name)
   - **Type:** Choose based on your needs:
     - **Regional:** Single region, lower latency
     - **Global:** Multi-region replication (higher cost)
   - **Region:** Select closest to your users
     - For Ghana/Mali: `EU-West-1 (Ireland)` or `US-East-1 (Virginia)`
   - **TLS:** Enable (recommended for production)
   - **Eviction:** Enable (recommended to prevent memory issues)

3. Click **"Create"**

### 3. Get Your Credentials

Upstash provides two ways to connect:

#### Option A: Redis Client (Recommended)
**Best for:** Production, better performance

```env
REDIS_URL=redis://default:AbCd...XyZ@endpoint.upstash.io:6379
REDIS_PASSWORD=AbCd...XyZ
```

**Advantages:**
- Faster (native Redis protocol)
- Full Redis feature support
- Pub/sub works perfectly
- Connection pooling

**Connection format:**
```env
# From Upstash console, copy the "Redis URL"
REDIS_URL=redis://default:AbCd...XyZ@endpoint.upstash.io:6379
REDIS_TOKEN=AbCd...XyZ  # Same as password in URL
```

#### Option B: REST API
**Best for:** Serverless functions, simple use cases

```env
UPSTASH_REDIS_REST_URL=https://endpoint.upstash.io
UPSTASH_REDIS_REST_TOKEN=AbCd...XyZ
```

**Advantages:**
- No connection management
- Works from any HTTP client
- Easier for debugging (curl-able)

**For our project:** Use **Option A (Redis Client)** for production.

### 4. Configure Your App

1. **Copy credentials from Upstash console**

2. **Update `.env`**
   ```env
   # Upstash Redis (recommended)
   REDIS_URL=redis://default:your-token@endpoint.upstash.io:6379
   REDIS_TOKEN=your-token
   
   # Or use REST API (alternative)
   # UPSTASH_REDIS_REST_URL=https://endpoint.upstash.io
   # UPSTASH_REDIS_REST_TOKEN=your-token
   ```

3. **Test the connection**
   ```bash
   go run cmd/check/main.go
   ```

   You should see:
   ```
   ✓ Redis connected
   ✓ Redis SET command working
   ✓ Redis GET command working
   ✓ Redis DEL command working
   ```

---

## Upstash Configuration

### Connection String Format

**Standard format:**
```
redis://[username]:[password]@[endpoint]:[port]
```

**Upstash format:**
```
redis://default:AbCdEfGh1234567890@gusc1-worthy-fish-12345.upstash.io:6379
         ^^^^^^^^                 ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
         username                 endpoint (unique to your database)
                ^^^^^^^^^^^^^^^^^^
                token (from console)
```

**With TLS (recommended):**
```
rediss://default:AbCdEfGh1234567890@gusc1-worthy-fish-12345.upstash.io:6379
^^^^^^
use 'rediss' (note the extra 's')
```

Our Go client handles TLS automatically when the URL starts with `rediss://`.

---

## Features We Use

### 1. OTP Storage (Phase 1)

**Use case:** Store 6-digit OTP codes for phone verification

```go
// Store OTP (10 min expiry)
key := fmt.Sprintf("otp:%s", phone)
redis.Set(ctx, key, "123456", 10*time.Minute)

// Retrieve and verify
code, err := redis.Get(ctx, key)
if code == userInput {
    redis.Del(ctx, key)  // Single use
}
```

**Upstash charges:** ~2 commands per OTP (SET + GET/DEL)

### 2. Rate Limiting (Phase 1)

**Use case:** Limit OTP sends to 3 per 10 minutes

```go
// Increment counter
key := fmt.Sprintf("otp:rate:%s", phone)
count, err := redis.Incr(ctx, key)

if count == 1 {
    // First attempt, set expiry
    redis.Expire(ctx, key, 10*time.Minute)
}

if count > 3 {
    return errors.New("rate limit exceeded")
}
```

**Upstash charges:** ~2 commands per attempt (INCR + EXPIRE)

### 3. Live Location Cache (Phase 5)

**Use case:** Cache latest truck GPS coordinates

```go
// Update location
key := fmt.Sprintf("trip:%s:location", tripID)
data := fmt.Sprintf(`{"lat":%f,"lng":%f,"ts":"%s"}`, lat, lng, time.Now())
redis.Set(ctx, key, data, 1*time.Hour)

// Retrieve for live tracking
location, err := redis.Get(ctx, key)
```

**Upstash charges:** 1 command per GPS ping

### 4. Pub/Sub for Realtime Updates (Phase 5)

**Use case:** Broadcast location updates to shipper app

```go
// Publisher (driver app)
channel := fmt.Sprintf("trip:%s:updates", tripID)
redis.Publish(ctx, channel, locationData)

// Subscriber (shipper app via WebSocket)
pubsub := redis.Subscribe(ctx, channel)
for msg := range pubsub.Channel() {
    // Forward to WebSocket client
    websocket.Send(msg.Payload)
}
```

**Upstash charges:**
- PUBLISH: 1 command per update
- SUBSCRIBE: 1 command to subscribe, then free for messages

---

## Cost Management

### Free Tier Limits
- **Commands:** 10,000/day
- **Storage:** 256 MB
- **Max command size:** 1 MB
- **Concurrent connections:** 100
- **Bandwidth:** 1 GB/month

### Estimating Usage (Per Day)

**Phase 1 (Auth):**
```
OTP sends: 100 users × 2 attempts = 200 OTPs
Commands: 200 × 2 (SET + GET) = 400 commands/day
Cost: FREE (well within 10K limit)
```

**Phase 5 (GPS Tracking):**
```
Active trips: 50
GPS pings: 50 trips × 60 pings/hour × 10 hours = 30,000 pings
Location cache: 30,000 × 1 (SET) = 30,000 commands/day
Pub/sub: 30,000 × 1 (PUBLISH) = 30,000 commands/day
Total: ~60,000 commands/day

Cost: $10/month (Pay-as-you-go tier)
```

**At scale (500 trips/day):**
```
Commands: ~600,000/day
Cost: ~$50/month
```

### Staying Within Budget

**Tips:**
1. **Use appropriate TTLs** — Don't cache forever
2. **Batch operations** — Use MSET for multiple keys
3. **Monitor usage** — Check Upstash console daily
4. **Optimize GPS ping frequency** — 60s intervals, not 1s

### When to Upgrade

- **Free tier → Pay-as-you-go ($10/mo):** >10K commands/day
- **Pay-as-you-go → Pro ($60/mo base):** Need 99.99% uptime SLA
- **Pro → Enterprise:** Need dedicated support or multi-region

---

## Regional Selection

### Recommended Regions for Our Users

| User Location | Recommended Upstash Region | Latency |
|---|---|---|
| Ghana (Accra) | EU-West-1 (Ireland) | ~150ms |
| Mali (Bamako) | EU-West-1 (Ireland) | ~180ms |
| Nigeria | EU-West-1 (Ireland) | ~140ms |

**Why EU-West-1?**
- Closest region to West Africa
- Good fiber optic connectivity
- Lower latency than US regions

**Alternative:** US-East-1 (Virginia) if most traffic is from Ghana (better undersea cable connectivity).

### Global Database (Advanced)

For multi-region replication:
- Primary: EU-West-1
- Read replica: US-East-1
- Automatic failover

**Cost:** ~3x base price
**When to use:** >100K active users, need <50ms latency globally

---

## Security Best Practices

### 1. Use TLS in Production

```env
# Production (with TLS)
REDIS_URL=rediss://default:password@endpoint.upstash.io:6379
           ^^^^^^
           note the 's'

# Local development (no TLS)
REDIS_URL=redis://localhost:6379
```

### 2. Rotate Tokens Regularly

**Via Upstash Console:**
1. Database → Settings → Reset Password
2. Update `.env` with new `REDIS_TOKEN` immediately
3. Restart server

**Best practice:** Rotate every 90 days.

### 3. Use IP Allowlist (Optional)

**For production:**
1. Database → Settings → IP Allowlist
2. Add your server's IP addresses
3. Block all other traffic

**Trade-off:** Less flexible, but more secure.

### 4. Never Commit Credentials

```bash
# ✅ Good
.env          # In .gitignore
.env.example  # Committed (no real credentials)

# ❌ Bad
.env          # Committed by accident → credentials exposed
```

---

## Monitoring & Debugging

### Via Upstash Console

**Metrics Dashboard:**
- Commands per second
- Hit rate (cache efficiency)
- Latency (p50, p95, p99)
- Memory usage
- Error rate

**Data Browser:**
- View all keys
- Inspect values
- Delete keys manually
- Execute Redis commands

### Using Redis CLI

**Connect to Upstash:**
```bash
redis-cli -u redis://default:your-token@endpoint.upstash.io:6379 --tls
```

**Common commands:**
```bash
# List all keys
KEYS *

# Get a value
GET otp:+233201234567

# Check TTL
TTL otp:+233201234567

# Monitor all commands (debugging)
MONITOR

# Check memory usage
INFO memory
```

### Debugging Tips

**"Connection timeout"**
- Check firewall/VPN blocking Upstash IPs
- Verify TLS is enabled (use `rediss://`)
- Check token is correct in `.env`

**"Command not supported"**
- Upstash supports 200+ Redis commands
- Some advanced commands (e.g., modules) not available
- Check: https://upstash.com/docs/redis/features/rediscompatibility

**"Memory limit exceeded"**
- Enable eviction policy (Upstash console → Settings)
- Clean up old keys (set proper TTLs)
- Upgrade to higher tier

---

## Performance Tips

### 1. Use Connection Pooling (Our Setup)

```go
// Our code (already configured)
sqlDB.SetMaxIdleConns(10)   // Reuse connections
sqlDB.SetMaxOpenConns(100)  // Limit concurrent connections
```

**Why it matters:**
- TLS handshake is expensive (~100ms)
- Connection reuse = faster responses
- Upstash charges per command, not per connection

### 2. Use Pipelining for Batch Operations

```go
// ❌ Slow (3 round trips)
redis.Set(ctx, "key1", "val1", 0)
redis.Set(ctx, "key2", "val2", 0)
redis.Set(ctx, "key3", "val3", 0)

// ✅ Fast (1 round trip)
pipe := redis.Pipeline()
pipe.Set(ctx, "key1", "val1", 0)
pipe.Set(ctx, "key2", "val2", 0)
pipe.Set(ctx, "key3", "val3", 0)
pipe.Exec(ctx)
```

**Performance gain:** 3x faster for batch operations

### 3. Choose Appropriate Data Structures

| Use Case | Data Structure | Commands |
|---|---|---|
| OTP storage | String | SET, GET, DEL |
| Rate limiting | String (counter) | INCR, EXPIRE |
| Location cache | String (JSON) | SET, GET |
| Trip status | Hash | HSET, HGET, HGETALL |
| Active drivers list | Set | SADD, SREM, SMEMBERS |
| Leaderboard | Sorted Set | ZADD, ZRANGE |

---

## Local Development Alternative

If you need offline development:

**Option 1: Use Upstash**
- Works online only
- Free tier sufficient for dev
- Same environment as production

**Option 2: Use Docker**
```bash
docker-compose up -d
```

Update `.env`:
```env
REDIS_URL=redis://localhost:6379
REDIS_TOKEN=
```

**Trade-offs:**
- Docker: Works offline, but different from production
- Upstash: Always matches production, requires internet

**Recommendation:** Use Upstash for dev, Docker for offline emergencies.

---

## Migration from Traditional Redis

If you have existing Redis data:

**Option 1: Export and Import**
```bash
# Export from old Redis
redis-cli --rdb dump.rdb

# Import to Upstash (requires Upstash CLI or manual scripting)
```

**Option 2: Dual Write**
```go
// Write to both old and new Redis
oldRedis.Set(ctx, key, value, ttl)
upstashRedis.Set(ctx, key, value, ttl)

// Gradually migrate traffic
// Then remove old Redis
```

**Time estimate:** 1-2 hours for small datasets (<1M keys)

---

## Useful Links

- **Upstash Console:** https://console.upstash.com
- **Upstash Documentation:** https://upstash.com/docs/redis
- **Redis Compatibility:** https://upstash.com/docs/redis/features/rediscompatibility
- **Pricing Calculator:** https://upstash.com/pricing
- **Status Page:** https://upstash.com/status

---

**Last Updated:** 2026-08-11 (Phase 0)
**Next Update:** Phase 1 (add OTP storage patterns)
