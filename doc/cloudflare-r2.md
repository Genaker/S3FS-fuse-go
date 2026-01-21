# Using s3fs-go with Cloudflare R2

Cloudflare R2 is an S3-compatible object storage service that doesn't charge egress fees. s3fs-go can be used with R2 by specifying the R2 endpoint URL.

## Prerequisites

- Cloudflare R2 account
- R2 bucket created
- API token with R2 read/write permissions

## Getting R2 Credentials

### 1. Create an API Token

1. Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Go to **My Profile** → **API Tokens**
3. Click **Create Token**
4. Use the **Edit Cloudflare Workers** template or create a custom token with:
   - **Account** → **Cloudflare R2** → **Edit** permissions
5. Save the token (you'll need it for authentication)

### 2. Get Your Account ID

Your Account ID can be found in the Cloudflare Dashboard URL or in the right sidebar of your dashboard.

### 3. Get Your R2 Bucket Name

The bucket name is the name you gave when creating the R2 bucket.

## Configuration

### Option 1: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your-r2-access-key-id
export AWS_SECRET_ACCESS_KEY=your-r2-secret-access-key
export AWS_REGION=auto  # R2 uses 'auto' as region
```

### Option 2: Passwd File

Create a passwd file with your R2 credentials:

```bash
# Format: ACCESS_KEY_ID:SECRET_ACCESS_KEY
echo "your-r2-access-key-id:your-r2-secret-access-key" > ~/.passwd-r2
chmod 600 ~/.passwd-r2
```

## R2 Endpoint Format

The R2 endpoint follows this format:
```
https://<account-id>.r2.cloudflarestorage.com
```

You can find your account ID in the Cloudflare Dashboard.

## Mounting R2 Bucket

### Basic Mount

```bash
./s3fs -bucket your-bucket-name \
       -mountpoint /mnt/r2 \
       -region auto \
       -endpoint https://your-account-id.r2.cloudflarestorage.com
```

### With Passwd File

```bash
./s3fs -bucket your-bucket-name \
       -mountpoint /mnt/r2 \
       -region auto \
       -endpoint https://your-account-id.r2.cloudflarestorage.com \
       -passwd_file ~/.passwd-r2
```

### With Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your-r2-access-key-id
export AWS_SECRET_ACCESS_KEY=your-r2-secret-access-key

./s3fs -bucket your-bucket-name \
       -mountpoint /mnt/r2 \
       -region auto \
       -endpoint https://your-account-id.r2.cloudflarestorage.com
```

## Complete Example

```bash
# 1. Set credentials
export AWS_ACCESS_KEY_ID=your-r2-access-key-id
export AWS_SECRET_ACCESS_KEY=your-r2-secret-access-key

# 2. Create mount point
sudo mkdir -p /mnt/r2

# 3. Mount R2 bucket
sudo ./s3fs -bucket my-r2-bucket \
            -mountpoint /mnt/r2 \
            -region auto \
            -endpoint https://abc123def456.r2.cloudflarestorage.com

# 4. Use the mounted filesystem
ls /mnt/r2
echo "Hello R2" > /mnt/r2/test.txt
cat /mnt/r2/test.txt

# 5. Unmount
sudo umount /mnt/r2
```

## Finding Your Account ID

1. Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Select your account
3. Your Account ID is displayed in the right sidebar
4. Or check the URL: `https://dash.cloudflare.com/<account-id>/...`

## Troubleshooting

### Error: "NoSuchBucket"

- Verify your bucket name is correct
- Ensure the bucket exists in your R2 account
- Check that your API token has access to the bucket

### Error: "InvalidAccessKeyId" or "SignatureDoesNotMatch"

- Verify your access key ID and secret access key are correct
- Ensure your API token has R2 permissions
- Check that credentials are properly exported or in the passwd file

### Error: "Access Denied"

- Verify your API token has R2 read/write permissions
- Check bucket permissions in Cloudflare Dashboard
- Ensure the bucket name matches exactly

### Connection Timeout

- Verify your endpoint URL is correct
- Check your network connection
- Ensure Cloudflare R2 service is accessible from your location

## R2-Specific Features

### No Egress Fees

Unlike AWS S3, Cloudflare R2 doesn't charge for data transfer (egress). This makes it ideal for high-traffic applications.

### S3-Compatible API

R2 uses the S3 API, so all s3fs-go features work with R2:
- File operations (create, read, write, delete)
- Directory operations
- Metadata operations
- Extended attributes
- Multi-part uploads

### Custom Domains

R2 supports custom domains. If you've configured a custom domain, you can use it as the endpoint:

```bash
./s3fs -bucket your-bucket-name \
       -mountpoint /mnt/r2 \
       -region auto \
       -endpoint https://your-custom-domain.com
```

## Comparison with AWS S3

| Feature | AWS S3 | Cloudflare R2 |
|---------|--------|---------------|
| Egress Fees | Yes | No |
| Storage Costs | Pay per GB | Pay per GB |
| API Compatibility | S3 API | S3 API |
| Regions | Multiple | Global |
| Endpoint Format | `s3.amazonaws.com` | `account-id.r2.cloudflarestorage.com` |

## Additional Resources

- [Cloudflare R2 Documentation](https://developers.cloudflare.com/r2/)
- [R2 API Reference](https://developers.cloudflare.com/r2/api/s3/api/)
- [Creating R2 API Tokens](https://developers.cloudflare.com/r2/api/s3/tokens/)
