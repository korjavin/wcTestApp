# WalletConnect Test Application

A Go implementation of the WalletConnect v2.0 protocol with a simplified relay server for educational purposes.

## Overview

This application demonstrates how to implement the WalletConnect v2.0 protocol in Go, allowing web applications to connect to Ethereum wallets like MetaMask or Trust Wallet without exposing private keys. The application includes:

1. A simplified relay server for message passing between dApps and wallets
2. A WalletConnect client implementation for establishing connections and requesting signatures
3. A simple web interface for connecting wallets and signing messages

## Features

- **Connect Wallet**: Generate a QR code that can be scanned by mobile wallets to establish a connection
- **Display Wallet Address**: Show the connected wallet's Ethereum address
- **Sign Messages**: Request the connected wallet to sign arbitrary messages
- **Educational Logging**: Detailed logs explaining the WalletConnect protocol flow

## Architecture

The application is structured as a single Go binary that serves both the web interface and implements the relay server functionality. The main components are:

1. **Relay Server**: A WebSocket-based server that handles message routing between dApps and wallets
2. **WalletConnect Client**: Implements the WalletConnect v2.0 protocol for pairing and session management
3. **Web Interface**: A simple HTML/CSS/JS interface for interacting with the application

## Prerequisites

- Go 1.24 or higher
- An Ethereum wallet that supports WalletConnect v2.0 (e.g., MetaMask, Trust Wallet)
- Docker and Docker Compose (optional, for containerized deployment)

## Important Note About WalletConnect and Public URLs

WalletConnect requires a publicly accessible URL for proper wallet connections. When running on localhost, some wallets may not be able to establish a connection. For production use, deploy the application to a server with a public domain name.

## Installation and Usage

### Local Development

Clone the repository and build the application:

```bash
git clone https://github.com/korjavin/wctestapp.git
cd wctestapp
go build -o wctestapp ./cmd/wctestapp
```

Run the application:

```bash
./wctestapp
```

Then open your browser and navigate to `http://localhost:8080` to access the web interface.

### Docker Deployment

The application can be run using Docker and Docker Compose:

```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f
```

### Environment Variables

The application can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| SERVER_HOST | Host to bind the HTTP server | 0.0.0.0 |
| SERVER_PORT | Port for the HTTP server | 8080 |
| SERVER_URL | External URL for the server (for QR codes) | http://localhost:8080 |
| RELAY_HOST | Host to bind the relay server | 0.0.0.0 |
| RELAY_PORT | Port for the relay server | 8081 |
| ENABLE_TLS | Enable HTTPS | false |
| CERT_FILE | Path to TLS certificate | certs/server.crt |
| KEY_FILE | Path to TLS private key | certs/server.key |
| DEBUG | Enable debug logging | true |

### HTTPS Setup

For production use, HTTPS is recommended and may be required by some wallets. There are two options for enabling HTTPS:

#### Option 1: Self-signed certificates (for testing only)

1. Generate self-signed certificates:

```bash
./scripts/generate-certs.sh
```

2. Run the application with HTTPS enabled:

```bash
ENABLE_TLS=true ./wctestapp
```

3. For Docker deployment with self-signed certificates:

```bash
ENABLE_TLS=true SERVER_URL=https://yourdomain.com docker-compose up -d
```

#### Option 2: Let's Encrypt certificates (recommended for production)

The application now includes Nginx as a reverse proxy with Let's Encrypt integration for automatic HTTPS:

1. Initialize Let's Encrypt certificates for your domain:

```bash
./scripts/init-letsencrypt.sh yourdomain.com your-email@example.com
```

2. Start the application with Docker Compose:

```bash
SERVER_URL=https://yourdomain.com docker-compose up -d
```

This setup will:
- Automatically obtain and renew Let's Encrypt certificates
- Handle HTTPS termination at the Nginx level
- Proxy both HTTP and WebSocket connections securely
- Redirect HTTP traffic to HTTPS

The certificates will be automatically renewed every 60 days.

## Using with a Public Domain

To use the application with a public domain:

1. Set up a server with a public IP address
2. Configure your domain to point to the server
3. Run the application with the appropriate SERVER_URL:

```bash
SERVER_URL=https://yourdomain.com ENABLE_TLS=true ./wctestapp
```

Or with Docker:

```bash
SERVER_URL=https://yourdomain.com ENABLE_TLS=true docker-compose up -d
```

## Connecting a Wallet

1. Click the "Connect Wallet" button on the web interface
2. Scan the displayed QR code with your Ethereum wallet app
3. Approve the connection request in your wallet
4. The web interface will display your connected wallet address

## Signing Messages

1. Enter a message in the text input field
2. Click the "Sign Message" button
3. Approve the signature request in your wallet
4. The web interface will display the resulting signature

## Educational Aspects

This application is designed for educational purposes and includes:

- Detailed logging of all operations with explanations
- Visualization of the connection and signing process
- Comprehensive code comments explaining implementation details
- Simplified relay server implementation for better understanding

## Project Structure

```
wctestapp/
├── cmd/wctestapp/         # Application entry point
├── internal/              # Internal packages
│   ├── config/            # Configuration handling
│   ├── relay/             # Relay server implementation
│   ├── wallet/            # WalletConnect client implementation
│   ├── server/            # HTTP server
│   └── logger/            # Logging utilities
├── web/                   # Web interface
│   ├── static/            # Static assets
│   └── templates/         # HTML templates
├── pkg/                   # Public packages
│   └── utils/             # Utility functions
├── scripts/               # Utility scripts
│   ├── generate-certs.sh  # Generate self-signed certificates
│   └── init-letsencrypt.sh # Initialize Let's Encrypt certificates
├── nginx/                 # Nginx configuration
│   └── nginx.conf         # Nginx configuration file
├── certbot/               # Let's Encrypt certificates (created at runtime)
│   ├── conf/              # Certificate configuration
│   └── www/               # ACME challenge files
├── Dockerfile             # Docker build instructions
└── docker-compose.yml     # Docker Compose configuration with Nginx and Certbot
```

## Building and Testing

Build the application:

```bash
go build -o wctestapp ./cmd/wctestapp
```

Run tests:

```bash
go test ./...
```

Build Docker image:

```bash
docker build -t wctestapp .
```

## Troubleshooting

### Connection Issues

If you're having trouble connecting your wallet:

1. Ensure you're using a public URL (not localhost) for production use
2. Check that your wallet supports WalletConnect v2.0
3. Enable HTTPS for better compatibility with wallets
4. Check the application logs for detailed error messages

### HTTPS Issues

If you're having trouble with HTTPS:

#### Self-signed certificates:
1. Ensure your certificates are valid and in the correct location
2. Check that the CERT_FILE and KEY_FILE environment variables are set correctly
3. For self-signed certificates, you may need to accept the security warning in your browser

#### Let's Encrypt with Nginx:
1. Make sure your domain is correctly pointing to your server's IP address
2. Check Nginx logs for certificate issues: `docker-compose logs nginx`
3. Check Certbot logs: `docker-compose logs certbot`
4. If certificates aren't being issued, try running the init script again with the production flag uncommented
5. Ensure ports 80 and 443 are open on your firewall and not blocked by your hosting provider

### WebSocket Connection Issues

If you're having trouble with WebSocket connections:

1. Ensure the WebSocket URL is using the correct protocol (ws:// or wss://)
2. With the Nginx proxy, all WebSocket connections should use wss:// protocol
3. Check browser console for WebSocket connection errors
4. Verify that the Nginx configuration is correctly proxying WebSocket connections

## Acknowledgements

- [WalletConnect](https://specs.walletconnect.com/2.0) for the protocol specification
- The Ethereum community for standards like EIP-191 and EIP-712