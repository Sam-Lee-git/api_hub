#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# AI API Proxy Platform — One-Click Ubuntu Deploy Script
# Tested on Ubuntu 22.04 / 24.04
# Usage: bash deploy.sh
# ============================================================

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
die()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ---- Must run as root or with sudo ----
[[ $EUID -ne 0 ]] && die "Please run as root: sudo bash deploy.sh"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$PROJECT_DIR/backend"
FRONTEND_DIR="$PROJECT_DIR/frontend"
DEPLOY_DIR="$SCRIPT_DIR"

GO_VERSION="1.22.4"
NODE_VERSION="22"
INSTALL_DIR="/opt/ai-proxy"

echo ""
echo "=============================================="
echo "   AI API Proxy Platform — Deploy Script"
echo "=============================================="
echo ""

# ============================================================
# 1. Collect configuration
# ============================================================
info "Collecting configuration (press Enter to accept defaults)..."
echo ""

read -rp "Domain name (e.g. api.example.com, or leave empty for IP-only): " DOMAIN
read -rp "Admin email [$USER@$(hostname)]: " ADMIN_EMAIL
ADMIN_EMAIL="${ADMIN_EMAIL:-$USER@$(hostname)}"
read -rsp "Admin password (min 12 chars): " ADMIN_PASSWORD; echo
[[ ${#ADMIN_PASSWORD} -lt 12 ]] && die "Admin password must be at least 12 characters"

echo ""
echo "--- AI Provider API Keys (leave blank to configure later) ---"
read -rp "OpenAI API key (sk-...): " OPENAI_API_KEY
read -rp "Anthropic API key (sk-ant-...): " ANTHROPIC_API_KEY
read -rp "Google API key (AIza...): " GOOGLE_API_KEY
read -rp "Alibaba/Qwen API key: " ALIBABA_API_KEY

echo ""
echo "--- Alipay credentials (leave blank to configure later) ---"
read -rp "Alipay App ID: " ALIPAY_APP_ID
read -rp "Alipay notify URL [https://${DOMAIN:-yourdomain.com}/api/payment/alipay/notify]: " ALIPAY_NOTIFY_URL
ALIPAY_NOTIFY_URL="${ALIPAY_NOTIFY_URL:-https://${DOMAIN:-yourdomain.com}/api/payment/alipay/notify}"

echo ""
echo "--- WeChat Pay credentials (leave blank to configure later) ---"
read -rp "WeChat Mch ID: " WECHAT_MCH_ID
read -rp "WeChat App ID: " WECHAT_APP_ID
read -rp "WeChat notify URL [https://${DOMAIN:-yourdomain.com}/api/payment/wechat/notify]: " WECHAT_NOTIFY_URL
WECHAT_NOTIFY_URL="${WECHAT_NOTIFY_URL:-https://${DOMAIN:-yourdomain.com}/api/payment/wechat/notify}"

# ---- Auto-generate secrets ----
gen_secret() { openssl rand -hex 32; }
DB_PASSWORD=$(gen_secret)
JWT_ACCESS_SECRET=$(gen_secret)
JWT_REFRESH_SECRET=$(gen_secret)
WECHAT_API_V3_KEY=$(openssl rand -hex 16)  # 32 hex chars = 32 bytes but WeChat needs 32 printable

info "Secrets auto-generated."

# ============================================================
# 2. Install system dependencies
# ============================================================
info "Updating apt..."
apt-get update -qq

install_if_missing() {
    dpkg -s "$1" &>/dev/null || apt-get install -y -qq "$1"
}

info "Installing base packages..."
for pkg in curl wget git unzip build-essential openssl nginx certbot python3-certbot-nginx; do
    install_if_missing "$pkg"
done

# ---- Go ----
if ! command -v go &>/dev/null || [[ "$(go version 2>/dev/null | awk '{print $3}')" != "go${GO_VERSION}" ]]; then
    info "Installing Go ${GO_VERSION}..."
    ARCH="$(uname -m)"
    [[ "$ARCH" == "x86_64" ]] && GOARCH="amd64" || GOARCH="arm64"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    ln -sf /usr/local/go/bin/go   /usr/local/bin/go
    ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
    ok "Go $(go version) installed"
else
    ok "Go already installed: $(go version)"
fi

export PATH=$PATH:/usr/local/go/bin

# ---- Node.js ----
if ! command -v node &>/dev/null || [[ "$(node -v | cut -dv -f2 | cut -d. -f1)" -lt "$NODE_VERSION" ]]; then
    info "Installing Node.js ${NODE_VERSION}..."
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - >/dev/null
    apt-get install -y -qq nodejs
    ok "Node.js $(node -v) installed"
else
    ok "Node.js already installed: $(node -v)"
fi

# ---- Docker ----
if ! command -v docker &>/dev/null; then
    info "Installing Docker..."
    curl -fsSL https://get.docker.com | sh >/dev/null
    systemctl enable --now docker
    ok "Docker installed"
else
    ok "Docker already installed: $(docker --version)"
fi

if ! command -v docker compose &>/dev/null 2>&1; then
    info "Installing Docker Compose plugin..."
    apt-get install -y -qq docker-compose-plugin
fi

# ---- golang-migrate ----
if ! command -v migrate &>/dev/null; then
    info "Installing golang-migrate..."
    MIGRATE_VERSION="4.17.1"
    MARCH="$(uname -m)"; [[ "$MARCH" == "x86_64" ]] && MARCH="amd64" || MARCH="arm64"
    wget -q "https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.linux-${MARCH}.tar.gz" -O /tmp/migrate.tar.gz
    tar -xzf /tmp/migrate.tar.gz -C /usr/local/bin migrate
    rm /tmp/migrate.tar.gz
    ok "migrate installed"
fi

# ============================================================
# 3. Create install directory and write .env
# ============================================================
info "Setting up $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"

ENV_FILE="$INSTALL_DIR/.env"
cat > "$ENV_FILE" <<EOF
DATABASE_URL=postgres://aiproxy:${DB_PASSWORD}@localhost:5432/aiproxy?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
OPENAI_API_KEY=${OPENAI_API_KEY}
ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
GOOGLE_API_KEY=${GOOGLE_API_KEY}
ALIBABA_API_KEY=${ALIBABA_API_KEY}
ALIPAY_APP_ID=${ALIPAY_APP_ID}
ALIPAY_NOTIFY_URL=${ALIPAY_NOTIFY_URL}
WECHAT_MCH_ID=${WECHAT_MCH_ID}
WECHAT_APP_ID=${WECHAT_APP_ID}
WECHAT_API_V3_KEY=${WECHAT_API_V3_KEY}
WECHAT_NOTIFY_URL=${WECHAT_NOTIFY_URL}
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
PORT=8080
ENV=production
EOF
chmod 600 "$ENV_FILE"
ok ".env written to $ENV_FILE"

# ============================================================
# 4. Start PostgreSQL + Redis via Docker Compose
# ============================================================
info "Starting PostgreSQL and Redis..."

# Write a minimal docker-compose for databases only (if deploy/docker-compose.yml references images)
DB_COMPOSE="$INSTALL_DIR/docker-compose-db.yml"
cat > "$DB_COMPOSE" <<YAML
version: "3.9"
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: aiproxy
      POSTGRES_PASSWORD: "${DB_PASSWORD}"
      POSTGRES_DB: aiproxy
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_data:/data

volumes:
  pg_data:
  redis_data:
YAML

docker compose -f "$DB_COMPOSE" up -d

info "Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
    docker exec "$(docker compose -f "$DB_COMPOSE" ps -q postgres)" pg_isready -U aiproxy &>/dev/null && break
    sleep 2
done
ok "PostgreSQL is ready"

# ============================================================
# 5. Run database migrations
# ============================================================
info "Running database migrations..."
export POSTGRESQL_URL="postgres://aiproxy:${DB_PASSWORD}@localhost:5432/aiproxy?sslmode=disable"
migrate -path "$BACKEND_DIR/internal/db/migrations" -database "$POSTGRESQL_URL" up
ok "Migrations applied"

# ============================================================
# 6. Build and install backend
# ============================================================
info "Building Go backend..."
cd "$BACKEND_DIR"
go mod tidy
go build -o "$INSTALL_DIR/backend" ./cmd/server/main.go
ok "Backend binary built at $INSTALL_DIR/backend"

# ---- Backend systemd service ----
cat > /etc/systemd/system/ai-proxy-backend.service <<UNIT
[Unit]
Description=AI Proxy Platform Backend
After=network.target docker.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${INSTALL_DIR}/backend
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable ai-proxy-backend
systemctl restart ai-proxy-backend
ok "Backend service started"

# ============================================================
# 7. Build and install frontend
# ============================================================
info "Building Next.js frontend..."
cd "$FRONTEND_DIR"
npm ci --silent
npm run build

FRONTEND_STANDALONE="$FRONTEND_DIR/.next/standalone"
FRONTEND_DEST="$INSTALL_DIR/frontend"

rm -rf "$FRONTEND_DEST"
cp -r "$FRONTEND_STANDALONE" "$FRONTEND_DEST"
cp -r "$FRONTEND_DIR/.next/static"  "$FRONTEND_DEST/.next/static"
cp -r "$FRONTEND_DIR/public"        "$FRONTEND_DEST/public"

ok "Frontend built at $FRONTEND_DEST"

# ---- Frontend systemd service ----
cat > /etc/systemd/system/ai-proxy-frontend.service <<UNIT
[Unit]
Description=AI Proxy Platform Frontend
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${FRONTEND_DEST}
Environment=NODE_ENV=production
Environment=PORT=3000
Environment=HOSTNAME=127.0.0.1
ExecStart=/usr/bin/node server.js
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable ai-proxy-frontend
systemctl restart ai-proxy-frontend
ok "Frontend service started"

# ============================================================
# 8. Configure Nginx
# ============================================================
info "Configuring Nginx..."

SERVER_NAME="${DOMAIN:-_}"
NGINX_CONF="/etc/nginx/sites-available/ai-proxy"

cat > "$NGINX_CONF" <<NGINX
upstream backend  { server 127.0.0.1:8080; }
upstream frontend { server 127.0.0.1:3000; }

server {
    listen 80;
    server_name ${SERVER_NAME};

    # SSE / streaming — must disable buffering
    proxy_buffering          off;
    proxy_cache              off;
    proxy_read_timeout       330s;
    proxy_send_timeout       330s;
    proxy_connect_timeout    10s;

    proxy_set_header Host              \$host;
    proxy_set_header X-Real-IP         \$remote_addr;
    proxy_set_header X-Forwarded-For   \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;

    # Backend: proxy API + v1 routes
    location ~ ^/(api|v1|health)/ {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    # Frontend: all other routes
    location / {
        proxy_pass http://frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade    \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
NGINX

ln -sf "$NGINX_CONF" /etc/nginx/sites-enabled/ai-proxy
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
ok "Nginx configured"

# ============================================================
# 9. Optional SSL via Certbot
# ============================================================
if [[ -n "$DOMAIN" ]]; then
    read -rp "Configure SSL with Let's Encrypt for $DOMAIN? [y/N]: " SSL_CONFIRM
    if [[ "${SSL_CONFIRM,,}" == "y" ]]; then
        info "Running certbot..."
        certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL"
        ok "SSL certificate issued"
    fi
fi

# ============================================================
# 10. Health check
# ============================================================
info "Running health check..."
sleep 3

BACKEND_STATUS=$(curl -sf http://127.0.0.1:8080/health && echo "ok" || echo "fail")
FRONTEND_STATUS=$(curl -sf http://127.0.0.1:3000 -o /dev/null -w "%{http_code}" || echo "000")

echo ""
echo "=============================================="
echo "   Deployment Complete"
echo "=============================================="
echo ""
[[ "$BACKEND_STATUS" == "ok" ]] \
    && ok  "Backend  → http://127.0.0.1:8080  ✓" \
    || warn "Backend  → http://127.0.0.1:8080  FAILED (check: journalctl -u ai-proxy-backend -n 50)"

[[ "$FRONTEND_STATUS" =~ ^2 ]] \
    && ok  "Frontend → http://127.0.0.1:3000  ✓" \
    || warn "Frontend → http://127.0.0.1:3000  HTTP $FRONTEND_STATUS (check: journalctl -u ai-proxy-frontend -n 50)"

echo ""
if [[ -n "$DOMAIN" ]]; then
    echo -e "  User portal : ${GREEN}https://${DOMAIN}${NC}"
    echo -e "  Admin panel : ${GREEN}https://${DOMAIN}/admin/dashboard${NC}"
    echo -e "  API endpoint: ${GREEN}https://${DOMAIN}/v1/chat/completions${NC}"
else
    SERVER_IP=$(curl -sf https://api.ipify.org || hostname -I | awk '{print $1}')
    echo -e "  User portal : ${GREEN}http://${SERVER_IP}${NC}"
    echo -e "  Admin panel : ${GREEN}http://${SERVER_IP}/admin/dashboard${NC}"
    echo -e "  API endpoint: ${GREEN}http://${SERVER_IP}/v1/chat/completions${NC}"
fi
echo ""
echo -e "  Admin email   : ${CYAN}${ADMIN_EMAIL}${NC}"
echo -e "  .env location : ${CYAN}${ENV_FILE}${NC}"
echo ""
echo "  Useful commands:"
echo "    journalctl -fu ai-proxy-backend    # backend logs"
echo "    journalctl -fu ai-proxy-frontend   # frontend logs"
echo "    systemctl restart ai-proxy-backend # restart backend"
echo "    docker compose -f $DB_COMPOSE ps  # db status"
echo ""
ok "Done."
