#!/bin/sh
set -e

ENDPOINT="${ENX_OBJECT_STORAGE_ENDPOINT:-minio:9000}"
BUCKET="${ENX_OBJECT_STORAGE_BUCKET:-envnexus}"
ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"

echo "Configuring MinIO alias (endpoint: ${ENDPOINT})..."
mc alias set enx "http://${ENDPOINT}" "${ACCESS_KEY}" "${SECRET_KEY}" --api S3v4

echo "Ensuring bucket exists..."
mc mb --ignore-existing "enx/${BUCKET}"

echo "Uploading agent base packages..."
for f in /agents/enx-agent-*; do
    name=$(basename "$f")
    echo "  Uploading ${name}..."
    mc cp "$f" "enx/${BUCKET}/base-packages/${name}"
    echo "  ✓ ${name}"
done

echo ""
echo "Verifying uploads:"
mc ls "enx/${BUCKET}/base-packages/"

echo ""
echo "All agent base packages uploaded successfully!"
