#!/bin/bash
# Script to generate self-signed certificates for HTTPS

# Create certs directory if it doesn't exist
mkdir -p certs

# Generate private key
openssl genrsa -out certs/server.key 2048

# Generate certificate signing request
# You can modify the subject to match your domain
openssl req -new -key certs/server.key -out certs/server.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Generate self-signed certificate (valid for 365 days)
openssl x509 -req -days 365 -in certs/server.csr -signkey certs/server.key -out certs/server.crt

# Remove the CSR as it's no longer needed
rm certs/server.csr

# Set permissions
chmod 600 certs/server.key
chmod 644 certs/server.crt

echo "Self-signed certificates generated in the certs directory."
echo "To use with a real domain, update the CN in the script and regenerate."