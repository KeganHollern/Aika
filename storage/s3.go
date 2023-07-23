package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3 struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
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
