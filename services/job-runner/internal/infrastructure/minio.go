package infrastructure

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client     *minio.Client
	bucketName string
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

func (m *MinIOClient) Client() *minio.Client {
	return m.client
}

func (m *MinIOClient) BucketName() string {
	return m.bucketName
}
