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
	client         *minio.Client
	bucketName     string
	publicEndpoint string
	publicUseSSL   bool
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

// SetPublicEndpoint configures an externally-accessible endpoint for presigned URLs.
// When set, presigned URLs will have their host rewritten from the internal Docker
// endpoint (e.g. minio:9000) to this public one (e.g. 192.168.1.100:9000).
func (m *MinIOClient) SetPublicEndpoint(endpoint string, useSSL bool) {
	m.publicEndpoint = endpoint
	m.publicUseSSL = useSSL
	slog.Info("MinIO public endpoint configured", "public_endpoint", endpoint, "ssl", useSSL)
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
	u, err := m.client.PresignedGetObject(ctx, m.bucketName, objectName, expiry, reqParams)
	if err != nil {
		return nil, err
	}

	if m.publicEndpoint != "" {
		scheme := "http"
		if m.publicUseSSL {
			scheme = "https"
		}
		u.Scheme = scheme
		u.Host = m.publicEndpoint
	}

	return u, nil
}
