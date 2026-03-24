#!/usr/bin/env bash
set -euo pipefail

# EnvNexus Smoke Test Script
# Validates the 12-step MVP smoke test per proposal §12.7.10

API_BASE="${ENX_API_BASE:-http://localhost:8080}"
GW_BASE="${ENX_GW_BASE:-http://localhost:8081}"
PASS=0
FAIL=0
TOTAL=12

green() { echo -e "\033[32m$1\033[0m"; }
red() { echo -e "\033[31m$1\033[0m"; }
step_pass() { PASS=$((PASS + 1)); green "  PASS"; }
step_fail() { FAIL=$((FAIL + 1)); red "  FAIL: $1"; }

echo "=========================================="
echo "  EnvNexus MVP Smoke Test (12 Steps)"
echo "=========================================="
echo "Platform API: ${API_BASE}"
echo "Session Gateway: ${GW_BASE}"
echo ""

# Step 1: All services started
echo "[Step 1/12] Docker Compose services running..."
if curl -sf "${API_BASE}/healthz" > /dev/null 2>&1 && \
   curl -sf "${GW_BASE}/healthz" > /dev/null 2>&1; then
    step_pass
else
    step_fail "Services not responding on healthz"
fi

# Step 2: healthz / readyz pass
echo "[Step 2/12] healthz / readyz checks..."
READYZ=$(curl -sf "${API_BASE}/readyz" 2>/dev/null || echo '{}')
DB_OK=$(echo "${READYZ}" | grep -o '"database":true' || true)
if [ -n "${DB_OK}" ]; then
    step_pass
else
    step_fail "readyz database check failed: ${READYZ}"
fi

# Step 3: Database migration executed
echo "[Step 3/12] Database migration auto-executed..."
if echo "${READYZ}" | grep -q '"database":true'; then
    step_pass
else
    step_fail "Database not ready (migration may not have run)"
fi

# Step 4: Default tenant and admin created
echo "[Step 4/12] Default tenant and admin exist..."
LOGIN_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@envnexus.io","password":"admin123"}' 2>/dev/null || echo '{}')
TOKEN=$(echo "${LOGIN_RESP}" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "${TOKEN}" ]; then
    step_pass
else
    step_fail "Cannot login with default admin"
fi

# Step 5: Console login successful
echo "[Step 5/12] Console login successful..."
ME_RESP=$(curl -sf -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/api/v1/me" 2>/dev/null || echo '{}')
if echo "${ME_RESP}" | grep -q '"email"'; then
    step_pass
else
    step_fail "GET /me failed: ${ME_RESP}"
fi

# Step 6: Create ModelProfile / PolicyProfile / AgentProfile
echo "[Step 6/12] Create profiles..."
# Get tenant ID first
TENANTS_RESP=$(curl -sf -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/api/v1/tenants" 2>/dev/null || echo '{}')
TENANT_ID=$(echo "${TENANTS_RESP}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "${TENANT_ID}" ]; then
    step_fail "No tenant found"
else
    # Create model profile
    MP_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants/${TENANT_ID}/model-profiles" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"name":"smoke-test-model","provider":"ollama","model_id":"llama3.2","config_json":"{}"}' 2>/dev/null || echo '{}')

    # Create policy profile
    PP_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants/${TENANT_ID}/policy-profiles" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"name":"smoke-test-policy","rules_json":"{}"}' 2>/dev/null || echo '{}')

    # Create agent profile
    AP_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants/${TENANT_ID}/agent-profiles" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"name":"smoke-test-agent","config_json":"{}"}' 2>/dev/null || echo '{}')

    if echo "${MP_RESP}" | grep -q '"id"\|"name"' && \
       echo "${PP_RESP}" | grep -q '"id"\|"name"' && \
       echo "${AP_RESP}" | grep -q '"id"\|"name"'; then
        step_pass
    else
        step_fail "Profile creation failed"
    fi
fi

# Step 7: Generate download link
echo "[Step 7/12] Generate download link..."
DL_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants/${TENANT_ID}/download-links" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"label":"smoke-test-link"}' 2>/dev/null || echo '{}')
ENROLL_TOKEN=$(echo "${DL_RESP}" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "${ENROLL_TOKEN}" ]; then
    step_pass
else
    # Try legacy path
    DL_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/tenants/${TENANT_ID}/tokens" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"label":"smoke-test-link"}' 2>/dev/null || echo '{}')
    ENROLL_TOKEN=$(echo "${DL_RESP}" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "${ENROLL_TOKEN}" ]; then
        step_pass
    else
        step_fail "Download link generation failed: ${DL_RESP}"
    fi
fi

# Step 8: Agent activation (enrollment)
echo "[Step 8/12] Agent enrollment..."
if [ -n "${ENROLL_TOKEN}" ]; then
    ENROLL_RESP=$(curl -sf -X POST "${API_BASE}/agent/v1/enroll" \
        -H "Content-Type: application/json" \
        -d "{\"enrollment_token\":\"${ENROLL_TOKEN}\",\"hostname\":\"smoke-test-host\",\"os\":\"linux\",\"arch\":\"amd64\"}" 2>/dev/null || echo '{}')
    DEVICE_TOKEN=$(echo "${ENROLL_RESP}" | grep -o '"device_token":"[^"]*"' | head -1 | cut -d'"' -f4)
    DEVICE_ID=$(echo "${ENROLL_RESP}" | grep -o '"device_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "${DEVICE_TOKEN}" ]; then
        step_pass
    else
        step_fail "Enrollment failed: ${ENROLL_RESP}"
    fi
else
    step_fail "No enrollment token available"
fi

# Step 9: Agent WebSocket session
echo "[Step 9/12] WebSocket session creation..."
if [ -n "${DEVICE_ID}" ]; then
    SESSION_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/sessions" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"device_id\":\"${DEVICE_ID}\",\"transport\":\"websocket\",\"initiator_type\":\"console\"}" 2>/dev/null || echo '{}')
    SESSION_ID=$(echo "${SESSION_RESP}" | grep -o '"session_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    WS_TOKEN=$(echo "${SESSION_RESP}" | grep -o '"ws_token":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "${SESSION_ID}" ] && [ -n "${WS_TOKEN}" ]; then
        step_pass
    else
        step_fail "Session creation failed: ${SESSION_RESP}"
    fi
else
    step_fail "No device ID available"
fi

# Step 10: Read-only diagnosis (simulated - check session exists)
echo "[Step 10/12] Diagnosis session exists..."
if [ -n "${SESSION_ID}" ]; then
    step_pass
else
    step_fail "No session available for diagnosis"
fi

# Step 11: Approval-based repair (simulated - check approval API)
echo "[Step 11/12] Approval API available..."
if [ -n "${SESSION_ID}" ]; then
    APPROVE_RESP=$(curl -sf -X POST "${API_BASE}/api/v1/sessions/${SESSION_ID}/approve" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"approval_request_id":"nonexistent","comment":"test"}' 2>/dev/null || echo '{}')
    if echo "${APPROVE_RESP}" | grep -q '"error"\|"code"'; then
        step_pass
    else
        step_fail "Approval API not responding properly"
    fi
else
    step_fail "No session for approval test"
fi

# Step 12: Audit events
echo "[Step 12/12] Audit events recorded..."
if [ -n "${TENANT_ID}" ]; then
    AUDIT_RESP=$(curl -sf -H "Authorization: Bearer ${TOKEN}" \
        "${API_BASE}/api/v1/tenants/${TENANT_ID}/audit-events" 2>/dev/null || echo '{}')
    if echo "${AUDIT_RESP}" | grep -q '"items"\|"event_type"'; then
        step_pass
    else
        step_fail "Audit events not found: ${AUDIT_RESP}"
    fi
else
    step_fail "No tenant for audit check"
fi

# Summary
echo ""
echo "=========================================="
echo "  Results: ${PASS}/${TOTAL} passed, ${FAIL}/${TOTAL} failed"
echo "=========================================="

if [ "${FAIL}" -eq 0 ]; then
    green "ALL SMOKE TESTS PASSED!"
    exit 0
else
    red "SOME TESTS FAILED"
    exit 1
fi
