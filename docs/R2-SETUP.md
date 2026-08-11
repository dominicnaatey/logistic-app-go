# Cloudflare R2 Setup Guide
**Object Storage for KYC Documents & File Uploads**

Last Updated: 2026-08-11

---

## Why Cloudflare R2?

**R2 is perfect for this project because:**
- ✅ **No egress fees** — Unlike S3, R2 doesn't charge for downloads
- ✅ **S3-compatible** — Works with existing AWS SDK
- ✅ **Global CDN** — Fast access from Ghana/Mali via Cloudflare's network
- ✅ **Generous free tier** — 10 GB storage free forever
- ✅ **Simple pricing** — $0.015/GB/month for storage
- ✅ **Public & private** — Support for both public URLs and presigned URLs

---

## What We'll Store in R2

| Type | Example | Access | Phase |
|---|---|---|---|
| KYC Documents | Driver's license, ID cards | Private (presigned URLs) | Phase 1-2 |
| Vehicle Documents | Insurance, roadworthy cert | Private (presigned URLs) | Phase 2 |
| Customs Documents | Waybills, invoices | Private (presigned URLs) | Phase 3 |
| Company Registration | Business registration docs | Private (presigned URLs) | Phase 2 |

**Storage estimate:**
- 1,000 drivers × 3 docs × 2 MB = 6 GB
- Well within free tier!

---

## Getting Started with R2

### 1. Create a Cloudflare Account

1. Go to **https://dash.cloudflare.com/sign-up**
2. Sign up (free plan available)
3. Verify your email

### 2. Enable R2

1. In Cloudflare dashboard, click **R2** in the sidebar
2. Click **"Purchase R2"** (don't worry, free tier is generous)
3. Add payment method (required, but won't be charged within free tier)

###3. Create Your R2 Bucket

1. Click **"Create bucket"**
2. Configure:
   - **Bucket name:** `logistic-app-files` (or your preferred name)
     - Must be globally unique
     - Lowercase, numbers, hyphens only
   - **Location:** Automatic (Cloudflare chooses optimal location)
3. Click **"Create bucket"**

### 4. Create API Tokens

1. Go to **R2 → Manage R2 API Tokens**
2. Click **"Create API Token"**
3. Configure:
   - **Token name:** `logistic-app-backend`
   - **Permissions:** 
     - ✅ Object Read & Write
     - ✅ Admin Read & Write (for bucket operations)
   - **TTL:** Never expire (or set expiry if preferred)
4. Click **"Create API Token"**

5. **Save these immediately** (shown only once):
   ```
   Access Key ID: abc123def456...
   Secret Access Key: xyz789uvw012...
   ```

### 5. Get Your Account ID

1. In R2 dashboard, look for **Account ID** at the top
2. Copy it (format: `1234567890abcdef...`)

### 6. Configure Your App

Update `.env`:
```env
# Cloudflare R2
R2_ACCOUNT_ID=your-account-id-here
R2_ACCESS_KEY_ID=your-access-key-id-here
R2_SECRET_ACCESS_KEY=your-secret-access-key-here
R2_BUCKET_NAME=logistic-app-files
R2_PUBLIC_URL=https://logistic-app-files.your-account.r2.dev
```

**Getting R2_PUBLIC_URL:**
1. Go to your bucket in R2 dashboard
2. Click **Settings** → **Public Access**
3. Enable **"Allow Access"**
4. Copy the public URL shown
5. Or use custom domain (see below)

### 7. Test the Connection

```bash
go run cmd/check/main.go
```

Expected output:
```
☁️  Checking Cloudflare R2...
  ✓ R2 client initialized
  ✓ R2 bucket 'logistic-app-files' accessible
  ✓ R2 connection successful
```

---

## R2 Configuration Options

### Public vs Private Access

**Public Bucket (for public files):**
```env
# Anyone can download files via direct URL
R2_PUBLIC_URL=https://logistic-app-files.your-account.r2.dev
```

**Private Bucket (recommended for KYC docs):**
- Disable public access in R2 dashboard
- Use presigned URLs for temporary access
- Our code generates presigned URLs automatically

**Best Practice for Our Project:**
- Keep bucket private
- Use presigned URLs with 1-hour expiry
- Share URLs only with authorized users

### Custom Domain (Optional)

Instead of `*.r2.dev`, use your own domain:

1. **In Cloudflare R2:**
   - Bucket → Settings → Custom Domains
   - Click **"Connect Domain"**
   - Enter: `files.yourapp.com`

2. **Update `.env`:**
   ```env
   R2_PUBLIC_URL=https://files.yourapp.com
   ```

**Benefits:**
- Professional URLs
- Better branding
- HTTPS automatic via Cloudflare

---

## Using R2 in Code

### Upload a File

```go
import "logistic-app-go/pkg/storage"

// Initialize R2 client
r2Client, err := storage.NewR2Client(storage.R2Config{
    AccountID:   cfg.Storage.R2AccountID,
    AccessKeyID: cfg.Storage.R2AccessKeyID,
    SecretKey:   cfg.Storage.R2SecretAccessKey,
    BucketName:  cfg.Storage.R2BucketName,
    PublicURL:   cfg.Storage.R2PublicURL,
})

// Upload a file
file, _ := os.Open("license.jpg")
defer file.Close()

url, err := r2Client.Upload(
    ctx,
    "kyc/driver-123/license.jpg",  // key (path in bucket)
    file,                            // file content
    "image/jpeg",                    // content type
)

// url = "https://logistic-app-files.r2.dev/kyc/driver-123/license.jpg"
```

### Generate Presigned URL (Private Access)

```go
// Generate URL valid for 1 hour
presignedURL, err := r2Client.GetPresignedURL(
    ctx,
    "kyc/driver-123/license.jpg",
    1*time.Hour,
)

// Share presignedURL with authorized user
// URL expires after 1 hour
```

### Delete a File

```go
err := r2Client.Delete(ctx, "kyc/driver-123/license.jpg")
```

### List Files

```go
// List all files in a folder
files, err := r2Client.List(ctx, "kyc/driver-123/")

// files = ["kyc/driver-123/license.jpg", "kyc/driver-123/id.jpg", ...]
```

---

## File Organization

### Recommended Folder Structure

```
bucket/
├── kyc/
│   ├── drivers/
│   │   ├── {driver-id}/
│   │   │   ├── license.jpg
│   │   │   ├── national-id.jpg
│   │   │   └── photo.jpg
│   ├── fleets/
│   │   ├── {fleet-id}/
│   │   │   ├── business-registration.pdf
│   │   │   └── tax-certificate.pdf
│   └── vehicles/
│       ├── {truck-id}/
│       │   ├── insurance.pdf
│       │   ├── roadworthy-cert.pdf
│       │   └── registration.pdf
├── customs/
│   ├── {trip-id}/
│   │   ├── waybill.pdf
│   │   ├── invoice.pdf
│   │   └── transit-doc.pdf
└── temp/
    └── {upload-session-id}/
        └── pending-file.jpg
```

### File Naming Conventions

**Pattern:** `{category}/{entity-type}/{entity-id}/{document-type}.{ext}`

**Examples:**
- `kyc/drivers/uuid-123/license.jpg`
- `kyc/fleets/uuid-456/registration.pdf`
- `customs/trip-789/waybill.pdf`

**Benefits:**
- Easy to organize
- Easy to delete (just delete folder)
- Easy to list (prefix search)

---

## Security Best Practices

### 1. Never Expose Secret Keys

```bash
# ✅ Good - in .env (gitignored)
R2_SECRET_ACCESS_KEY=abc123...

# ❌ Bad - hardcoded in code
const secretKey = "abc123..."
```

### 2. Use Presigned URLs for Sensitive Files

```go
// ✅ Good - temporary access
presignedURL, _ := r2Client.GetPresignedURL(ctx, key, 1*time.Hour)

// ❌ Bad - permanent public access
publicURL := fmt.Sprintf("%s/%s", r2PublicURL, key)
```

### 3. Validate File Types

```go
// Check file extension
allowedExts := []string{".jpg", ".jpeg", ".png", ".pdf"}
ext := filepath.Ext(filename)
if !contains(allowedExts, strings.ToLower(ext)) {
    return errors.New("invalid file type")
}

// Check MIME type
contentType := http.DetectContentType(fileBytes)
if !strings.HasPrefix(contentType, "image/") && contentType != "application/pdf" {
    return errors.New("invalid content type")
}
```

### 4. Limit File Sizes

```go
// Max 5 MB per file
const maxFileSize = 5 * 1024 * 1024

if fileSize > maxFileSize {
    return errors.New("file too large")
}
```

### 5. Scan for Malware (Optional, Phase 10+)

Integrate with virus scanning service:
- Cloudflare Gateway (paid)
- ClamAV (open source)
- VirusTotal API

---

## Cost Management

### Free Tier
- **Storage:** 10 GB/month
- **Class A operations** (PUT, LIST): 1 million/month
- **Class B operations** (GET, HEAD): 10 million/month
- **Egress:** FREE (unlimited)

### Pricing (After Free Tier)
- **Storage:** $0.015/GB/month
- **Class A:** $4.50 per million requests
- **Class B:** $0.36 per million requests

### Cost Estimate (Our App)

**Storage:**
- 1,000 drivers × 3 docs × 2 MB = 6 GB
- Cost: FREE (within 10 GB tier)

**Operations (per month):**
- Uploads: 1,000 docs = 1,000 Class A requests
- Downloads: 10,000 views = 10,000 Class B requests
- Cost: FREE (well within limits)

**At scale (10,000 drivers):**
- Storage: 60 GB × $0.015 = $0.90/month
- Operations: Still FREE
- **Total: < $1/month**

---

## Monitoring & Debugging

### Via Cloudflare Dashboard

1. **Usage Metrics:**
   - Storage used
   - Requests per day
   - Bandwidth (though egress is free)

2. **Object Browser:**
   - View all files
   - Download files
   - Delete files manually

### Debugging Tips

**"Access Denied" Error:**
- Check API token permissions
- Verify bucket name is correct
- Check file key exists

**"Bucket Not Found":**
- Verify `R2_ACCOUNT_ID` is correct
- Check bucket name spelling
- Ensure bucket exists in your account

**"Unauthorized":**
- Regenerate API tokens
- Check `R2_ACCESS_KEY_ID` and `R2_SECRET_ACCESS_KEY`
- Verify token hasn't expired

---

## Useful Links

- **R2 Dashboard:** https://dash.cloudflare.com/r2
- **R2 Documentation:** https://developers.cloudflare.com/r2/
- **S3 API Compatibility:** https://developers.cloudflare.com/r2/api/s3/
- **Pricing Calculator:** https://developers.cloudflare.com/r2/pricing/

---

**Last Updated:** 2026-08-11 (Phase 0)
**Next Update:** Phase 1 (add file upload examples)
