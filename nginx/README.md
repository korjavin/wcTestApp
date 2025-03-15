# Nginx Configuration

This directory contains the Nginx configuration for the WalletConnect Test Application.

## Overview

The Nginx server acts as a reverse proxy in front of the Go application, providing:

1. **HTTPS Termination**: Handles TLS/SSL connections with Let's Encrypt certificates
2. **WebSocket Proxy**: Securely proxies WebSocket connections for the relay server
3. **HTTP to HTTPS Redirection**: Automatically redirects HTTP traffic to HTTPS
4. **ACME Challenge Support**: Supports Let's Encrypt certificate issuance and renewal

## Configuration

The `nginx.conf` file is configured to:

- Listen on ports 80 (HTTP) and 443 (HTTPS)
- Redirect all HTTP traffic to HTTPS
- Proxy HTTP requests to the Go application
- Proxy WebSocket connections to the Go application
- Support Let's Encrypt certificate challenges

## Usage

The Nginx configuration is automatically used when running the application with Docker Compose:

```bash
SERVER_URL=https://yourdomain.com docker-compose up -d
```

## Customization

If you need to customize the Nginx configuration:

1. Modify the `nginx.conf` file
2. Restart the Nginx container:

```bash
docker-compose restart nginx
```

## Troubleshooting

If you encounter issues with the Nginx configuration:

1. Check the Nginx logs:

```bash
docker-compose logs nginx
```

2. Verify that your domain is correctly pointing to your server
3. Ensure ports 80 and 443 are open on your firewall