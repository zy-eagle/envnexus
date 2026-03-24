#!/usr/bin/env bash
set -euo pipefail

# EnvNexus Seed Script
# Initializes default tenant and admin user via the platform API

API_BASE="${ENX_API_BASE:-http://localhost:8080}"

echo "=== EnvNexus Seed Script ==="
echo "API Base: ${API_BASE}"
echo ""

# Wait for API to be ready
echo "[1/4] Waiting for platform-api to be ready..."
for i in $(seq 1 30); do
    if curl -sf "${API_BASE}/healthz" > /dev/null 2>&1; then
        echo "  Platform API is healthy."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "  ERROR: Platform API not ready after 30 seconds."
        exit 1
    fi
    sleep 1
done

# Check readyz
echo "[2/4] Checking readiness..."
READYZ=$(curl -sf "${API_BASE}/readyz" 2>/dev/null || echo '{"status":"error"}')
echo "  readyz: ${READYZ}"

# Login with default admin
echo "[3/4] Logging in with default admin..."
LOGIN_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@envnexus.io","password":"admin123"}' 2>/dev/null || echo '{"error":"login failed"}')

TOKEN=$(echo "${LOGIN_RESP}" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "${TOKEN}" ]; then
    echo "  WARNING: Could not extract token. Login response: ${LOGIN_RESP}"
    echo "  The seed data may already exist from migration."
    exit 0
fi
echo "  Login successful, token obtained."

# Create default tenant if not exists
echo "[4/4] Checking/creating default tenant..."
TENANTS=$(curl -sf -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/api/v1/tenants" 2>/dev/null || echo '{"data":[]}')
TENANT_COUNT=$(echo "${TENANTS}" | grep -o '"id"' | wc -l)

if [ "${TENANT_COUNT}" -gt 0 ]; then
    echo "  Tenants already exist (count: ${TENANT_COUNT}). Skipping creation."
else
    CREATE_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"name":"Default Tenant","slug":"default","plan":"pro"}' 2>/dev/null || echo '{"error":"create failed"}')
    echo "  Create tenant response: ${CREATE_RESP}"
fi

echo ""
echo "=== Seed Complete ==="
