# OxBin WebUI SSH Deployment Commands

This document contains all the commands that were executed on the SSH server to deploy OxBin WebUI with Nginx reverse proxy and SSL certificate.

## Server Information
- **Domain**: oxbin.jeet22.xyz
- **User**: ubuntu
- **OS**: Ubuntu

---

## 1. Initial Server Setup

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install essential packages
sudo apt install -y curl wget git unzip software-properties-common

# Configure UFW firewall
sudo ufw enable
sudo ufw allow ssh
sudo ufw allow 80
sudo ufw allow 443

# Check firewall status
sudo ufw status
```

## 2. Application Deployment (Local Build Method)

### On Local Machine (Development Environment):
```bash
# Navigate to project directory
cd /path/to/oxbin

# Install templ if not already installed
go install github.com/a-h/templ/cmd/templ@latest

# Generate templ files
templ generate

# Build for Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -a -installsuffix cgo -o bin/oxbin-webui-linux ./cmd/webui

# Verify the binary was created
ls -la bin/oxbin-webui-linux
```

### Transfer Binary to VPS:
```bash
# Create application directory on VPS
ssh fluence "mkdir -p /home/ubuntu/oxbin/{bin,logs}"

# Transfer the binary using scp
scp bin/oxbin-webui-linux fluence:/home/ubuntu/oxbin/bin/

# Rename and set executable permissions
ssh fluence "cd /home/ubuntu/oxbin/bin && mv bin/oxbin-webui-linux oxbin-webui && chmod +x oxbin-webui"
```

## 3. Create Systemd Service

```bash
# Create systemd service file
sudo tee /etc/systemd/system/oxbin-webui.service > /dev/null << 'EOF'
[Unit]
Description=OxBin WebUI Service
After=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu/oxbin/
ExecStart=/home/ubuntu/oxbin/bin/oxbin-webui
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
Environment=PORT=39293

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd configuration
sudo systemctl daemon-reload

# Enable and start the service
sudo systemctl enable oxbin-webui
sudo systemctl start oxbin-webui

# Check service status
sudo systemctl status oxbin-webui
```

## 4. Test Binary Manually (Troubleshooting)

```bash
# Navigate to application directory
cd /home/ubuntu/oxbin

# Test running the binary directly
./oxbin-webui

# Expected output:
# 2025/09/27 16:31:10 🌐 OxBin Web UI starting on port 8080
# 2025/09/27 16:31:10 📍 Access at: http://localhost:8080

# Stop with Ctrl+C after confirming it works
```

## 5. Install and Configure Nginx

```bash
# Install Nginx
sudo apt update
sudo apt install -y nginx

# Enable and start Nginx
sudo systemctl enable nginx
sudo systemctl start nginx

# Check Nginx status
sudo systemctl status nginx
```

## 6. Configure Nginx for Domain

```bash
# Create Nginx configuration for subdomain
sudo tee /etc/nginx/sites-available/oxbin > /dev/null << 'EOF'
server {
    listen 80;
    listen [::]:80;

    server_name oxbin.jeet22.xyz;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # Proxy settings
    location / {
        proxy_pass http://127.0.0.1:39293;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 86400;
    }

    # Increase client max body size for file uploads
    client_max_body_size 50M;

    # Logging
    access_log /var/log/nginx/oxbin_access.log;
    error_log /var/log/nginx/oxbin_error.log;
}
EOF

# Enable the site
sudo ln -sf /etc/nginx/sites-available/oxbin /etc/nginx/sites-enabled/

# Remove default site (optional)
sudo rm -f /etc/nginx/sites-enabled/default

# Test Nginx configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

## 7. Configure Firewall for Web Traffic

```bash
# Allow HTTP and HTTPS through firewall
sudo ufw allow 'Nginx Full'

# Or specifically:
sudo ufw allow 80
sudo ufw allow 443

# Check firewall status
sudo ufw status
```

## 8. Verification and Testing Commands

```bash
# Check all services are running
sudo systemctl status nginx
sudo systemctl status oxbin-webui

# Check SSL certificate
openssl s_client -connect oxbin.jeet22.xyz:443 -servername oxbin.jeet22.xyz

# View service logs
sudo journalctl -u oxbin-webui -f
sudo journalctl -u nginx -f

# Check Nginx logs
sudo tail -f /var/log/nginx/oxbin_access.log
sudo tail -f /var/log/nginx/oxbin_error.log
```

## 9. Maintenance Commands

```bash
# Restart services
sudo systemctl restart oxbin-webui
sudo systemctl restart nginx

# Reload Nginx configuration
sudo systemctl reload nginx

# Check service status
sudo systemctl status oxbin-webui --no-pager -l
sudo systemctl status nginx --no-pager -l

# View recent logs
sudo journalctl -u oxbin-webui -n 50
sudo journalctl -u nginx -n 50
```


## 10. Final Configuration Summary

### Service File Location:
```
/etc/systemd/system/oxbin-webui.service
```

### Nginx Configuration:
```
/etc/nginx/sites-available/oxbin
/etc/nginx/sites-enabled/oxbin (symlink)
```

### Application Files:
```
/home/ubuntu/oxbin/oxbin-webui (main binary)
/home/ubuntu/oxbin/logs/ (log directory)
```

### Log Files:
```
# Application logs
sudo journalctl -u oxbin-webui

# Nginx logs
/var/log/nginx/oxbin_access.log
/var/log/nginx/oxbin_error.log

# System logs
/var/log/syslog
```
---

*This deployment was completed successfully on September 27, 2025*
