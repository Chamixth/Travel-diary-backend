package integrations

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client  *s3.Client
	Presign *s3.PresignClient
	Bucket  string
	Region  string
}

func NewS3Client(ctx context.Context, region, accessKeyID, secretAccessKey, bucket string) (*S3Client, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		// Some S3 responses do not advertise a checksum. Validate checksums
		// when S3 requires one without warning for unsupported optional checks.
		awscfg.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	return &S3Client{
		Client:  client,
		Presign: s3.NewPresignClient(client),
		Bucket:  bucket,
		Region:  region,
	}, nil
}

func (s *S3Client) PublicURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.Bucket, s.Region, key)
}

// PresignGetObject returns a temporary HTTPS URL that any client can fetch
// without AWS credentials. We use this for OpenAI image analysis because the
// bucket itself remains private.
func (s *S3Client) PresignGetObject(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := s.Presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	return err
}

// PresignPutObject returns a URL the browser can PUT the file body to
// directly, with no bytes passing through our backend at all.
func (s *S3Client) PresignPutObject(ctx context.Context, key, contentType string, expires time.Duration) (string, error) {
	req, err := s.Presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.Bucket,
		Key:         &key,
		ContentType: &contentType,
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// PutObject uploads bytes the backend already holds in memory — used for
// generated derivatives (e.g. thumbnails) rather than the direct-from-browser
// original upload path, which goes through PresignPutObject instead.
func (s *S3Client) PutObject(ctx context.Context, key, contentType string, body []byte) error {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.Bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: &contentType,
	})
	return err
}

// HeadObject returns the authoritative size and content type S3 recorded for
// an object — used after a direct browser upload so we never have to trust
// client-reported metadata for what actually landed in the bucket.
func (s *S3Client) HeadObject(ctx context.Context, key string) (int64, string, error) {
	out, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.Bucket, Key: &key})
	if err != nil {
		return 0, "", err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	contentType := ""
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return size, contentType, nil
}

// GetObjectPrefix fetches at most maxBytes from the start of an object. Image
// dimensions and EXIF metadata live in the first few KB of a JPEG/PNG/GIF, so
// this avoids downloading a whole multi-megabyte photo just to read its
// header — the fallback in the caller re-fetches in full only if that prefix
// turns out to be insufficient to decode.
func (s *S3Client) GetObjectPrefix(ctx context.Context, key string, maxBytes int64) ([]byte, error) {
	rangeHeader := fmt.Sprintf("bytes=0-%d", maxBytes-1)
	out, err := s.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &key, Range: &rangeHeader})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

// GetObject fetches an entire object's bytes.
func (s *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := s.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}
