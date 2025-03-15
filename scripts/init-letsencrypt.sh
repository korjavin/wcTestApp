#!/bin/bash

# This script will initialize Let's Encrypt certificates for your domain
# Usage: ./init-letsencrypt.sh yourdomain.com [email@example.com]

if [ -z "$1" ]; then
  echo "Error: Please provide a domain name as the first argument"
  echo "Usage: $0 yourdomain.com [email@example.com]"
  exit 1
fi

domain=$1
email=${2:-""}
email_arg=""

if [ ! -z "$email" ]; then
  email_arg="--email $email"
fi

# Create required directories
mkdir -p ./certbot/conf/live/$domain
mkdir -p ./certbot/www

# Stop any running containers
docker-compose down

# Update the Nginx configuration with the correct domain
sed -i "s/example.com/$domain/g" ./nginx/nginx.conf

# Start Nginx
docker-compose up -d nginx

# Get certificates (staging)
docker-compose run --rm certbot certonly --webroot -w /var/www/certbot \
  $email_arg --agree-tos --no-eff-email \
  -d $domain --staging

# Get certificates (production)
# Uncomment the following lines when you're ready for production certificates
# docker-compose run --rm certbot certonly --webroot -w /var/www/certbot \
#   $email_arg --agree-tos --no-eff-email \
#   -d $domain --force-renewal

# Restart Nginx to load the certificates
docker-compose restart nginx

echo "Initialization completed!"
echo "To use production certificates, uncomment the production section in this script and run it again."