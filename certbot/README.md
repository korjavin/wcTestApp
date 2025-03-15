# Certbot Configuration

This directory is used by Certbot for Let's Encrypt certificate management.

## Overview

Certbot is used to automatically obtain and renew SSL/TLS certificates from Let's Encrypt. This directory contains:

1. **conf/**: Stores the issued certificates and configuration
2. **www/**: Used for the ACME challenge during certificate issuance and renewal

## Usage

The certificates are automatically managed when using the provided scripts and Docker Compose configuration:

```bash
# Initialize certificates for your domain
./scripts/init-letsencrypt.sh yourdomain.com your-email@example.com

# Start the application with Docker Compose
SERVER_URL=https://yourdomain.com docker-compose up -d
```

## Certificate Renewal

Certificates are automatically renewed by the Certbot container. The renewal process:

1. Runs every 12 hours (as configured in docker-compose.yml)
2. Only renews certificates that are close to expiration (within 30 days)
3. Uses the same validation method as the initial issuance

## Manual Certificate Management

If you need to manually manage certificates:

```bash
# Check certificate status
docker-compose run --rm certbot certificates

# Force renewal of a certificate
docker-compose run --rm certbot renew --force-renewal

# Issue a new certificate
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot \
  --email your-email@example.com --agree-tos --no-eff-email \
  -d yourdomain.com
```

## Troubleshooting

If you encounter issues with certificate issuance or renewal:

1. Check the Certbot logs:

```bash
docker-compose logs certbot
```

2. Verify that your domain is correctly pointing to your server
3. Ensure port 80 is open and accessible for the ACME challenge
4. Check that the Nginx configuration is correctly serving the ACME challenge files