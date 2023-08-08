package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
)

var (
	ErrNoDataTransfered = errors.New("no data transfered")
)

type S3 struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	PublicUrl string
}

func NewS3FromEnv() (*S3, error) {
	endpoint, exists := os.LookupEnv("S3_HOSTNAME")
	if !exists {
		return nil, fmt.Errorf("missing env var S3_HOSTNAME")
	}
	publicurl, exists := os.LookupEnv("S3_PUBLICURL")
	if !exists {
		return nil, fmt.Errorf("missing env var S3_PUBLICURL")
	}
	region, exists := os.LookupEnv("S3_REGION")
	if !exists {
		region = "auto"
	}
	access, exists := os.LookupEnv("S3_ACCESS")
	if !exists {
		return nil, fmt.Errorf("missing env var S3_ACCESS")
	}
	secret, exists := os.LookupEnv("S3_SECRET")
	if !exists {
		return nil, fmt.Errorf("missing env var S3_SECRET")
	}
	bucket, exists := os.LookupEnv("S3_BUCKET")
	if !exists {
		return nil, fmt.Errorf("missing env var S3_BUCKET")
	}

	logrus.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"region":   region,
		"access":   access[:4],
		"secret":   secret[:4],
		"public":   publicurl,
		"bucket":   bucket,
	}).Infoln("s3 configuration")

	return &S3{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: access,
		SecretKey: secret,
		Bucket:    bucket,
		PublicUrl: publicurl,
	}, nil
}

// DownloadAndUpload will download a generic file from a URL
// and upload that file to the provided key in S3
func (s *S3) DownloadAndUpload(url, key string) error {
	// Download the image
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from url; %w", err)
	}
	defer resp.Body.Close()

	// Create a buffer to copy the response
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("failed to read response body; %w", err)
	}

	// Create a new session with the custom endpoint and credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.Region),
		Endpoint:    aws.String(s.Endpoint),
		Credentials: credentials.NewStaticCredentials(s.AccessKey, s.SecretKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to s3; %w", err)
	}

	// Create a new S3 service
	s3Svc := s3.New(sess)

	// Upload the image to S3
	_, err = s3Svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("failed putobject; %w", err)
	}

	return nil
}

// StreamUpload streams data to S3 in chunks.
// This reduces memory and disk usage.
func (s *S3) StreamUpload(stream io.ReadCloser, key string) error {
	// Create a new session with the custom endpoint and credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.Region),
		Endpoint:    aws.String(s.Endpoint),
		Credentials: credentials.NewStaticCredentials(s.AccessKey, s.SecretKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to s3; %w", err)
	}

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   stream,
	}, func(u *s3manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 10MB part size
		u.LeavePartsOnError = false   // on fail delete garbage
	})
	if err != nil {
		return fmt.Errorf("failed putobject; %w", err)
	}

	exists, err := s.KeyExists(key)
	if err != nil {
		return fmt.Errorf("failed to check put succeeded; %w", err)
	}

	if !exists {
		return ErrNoDataTransfered
	}

	return nil
}

func (s *S3) KeyExists(key string) (bool, error) {
	// Create a new session with the custom endpoint and credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.Region),
		Endpoint:    aws.String(s.Endpoint),
		Credentials: credentials.NewStaticCredentials(s.AccessKey, s.SecretKey, ""),
	})
	if err != nil {
		return false, fmt.Errorf("failed to connect to s3; %w", err)
	}

	// Create a new S3 service
	s3Svc := s3.New(sess)

	out, err := s3Svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			code := aerr.Code()
			switch code {
			case s3.ErrCodeNoSuchKey, "NotFound": // couldn't find a "NotFound" and that's what i need lmfao
				return false, nil
			default:
				return false, fmt.Errorf("failed to headobject; %w", err)
			}
		}
		return false, fmt.Errorf("failed to headobject not a awserr; %w", err)
	}
	// don't count a key as 'existing' if its 0 bytes
	if out.ContentLength != nil && *out.ContentLength == 0 {
		return false, nil
	}

	return true, nil
}
