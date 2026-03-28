package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client       *minio.Client
	publicClient *minio.Client
	bucketName   string
}

func NewMinIOClient(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client creation failed: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check failed: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio bucket creation failed: %w", err)
		}
		slog.Info("created MinIO bucket", "bucket", bucketName)
	}

	slog.Info("connected to MinIO", "endpoint", endpoint, "bucket", bucketName)
	return &MinIOClient{client: client, bucketName: bucketName}, nil
}

// SetPublicEndpoint creates a separate MinIO client using the public endpoint
// for presigned URL generation. The signature is computed against the public host,
// so browsers can access the URL without SignatureDoesNotMatch errors.
func (m *MinIOClient) SetPublicEndpoint(endpoint, accessKey, secretKey string, useSSL bool) {
	publicClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		slog.Warn("failed to create public MinIO client, presigned URLs will use internal endpoint", "error", err)
		return
	}
	m.publicClient = publicClient
	slog.Info("MinIO public endpoint configured for presigned URLs", "public_endpoint", endpoint)
}

func (m *MinIOClient) Client() *minio.Client {
	return m.client
}

func (m *MinIOClient) BucketName() string {
	return m.bucketName
}

func (m *MinIOClient) Ping(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.bucketName)
	return err
}

func (m *MinIOClient) PresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (*url.URL, error) {
	reqParams := make(url.Values)
	c := m.client
	if m.publicClient != nil {
		c = m.publicClient
	}
	return c.PresignedGetObject(ctx, m.bucketName, objectName, expiry, reqParams)
}
