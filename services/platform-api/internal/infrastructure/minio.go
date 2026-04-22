package infrastructure

import (
	"context"
	"fmt"
	"io"
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

// PresignedDownloadURL generates a presigned GET URL with Content-Disposition: attachment
// so that browsers trigger a file download instead of displaying the content inline.
func (m *MinIOClient) PresignedDownloadURL(ctx context.Context, objectName string, expiry time.Duration, filename string) (*url.URL, error) {
	reqParams := make(url.Values)
	if filename != "" {
		reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	} else {
		reqParams.Set("response-content-disposition", "attachment")
	}
	c := m.client
	if m.publicClient != nil {
		c = m.publicClient
	}
	return c.PresignedGetObject(ctx, m.bucketName, objectName, expiry, reqParams)
}

// PresignedPutURL generates a presigned PUT URL for uploading an object.
// Uses the public client when available so agents running outside Docker
// can reach MinIO via a routable address (e.g. localhost:9000).
func (m *MinIOClient) PresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (*url.URL, error) {
	c := m.client
	if m.publicClient != nil {
		c = m.publicClient
	}
	return c.PresignedPutObject(ctx, m.bucketName, objectName, expiry)
}

// ObjectExists returns true if the object exists in the bucket.
func (m *MinIOClient) ObjectExists(ctx context.Context, objectName string) bool {
	_, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	return err == nil
}

// PutObject uploads an object to the configured bucket.
func (m *MinIOClient) PutObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, size, opts)
	return err
}

// RemoveObject deletes an object from the configured bucket. Missing objects are not treated as an error.
func (m *MinIOClient) RemoveObject(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err == nil {
		return nil
	}
	er := minio.ToErrorResponse(err)
	if er.Code == "NoSuchKey" || er.Code == "NotFound" {
		return nil
	}
	return err
}
