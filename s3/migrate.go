package s3

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

// DownloadImageAndUploadToS3 downloads an image from url and uploads to an S3 bucket
func (s3Config *S3Config) DownloadImageAndUploadToS3(url, key string) error {
	// Download the image
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a buffer to copy the response
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return err
	}

	// Create a new session with the custom endpoint and credentials
	logrus.WithField("cfg", s3Config).Info("S3 CONFIGURATION DUMP")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("US-East"),
		Endpoint:    aws.String(s3Config.Endpoint),
		Credentials: credentials.NewStaticCredentials(s3Config.AccessKey, s3Config.SecretKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to start session; %w", err)
	}

	// Create a new S3 service
	s3Svc := s3.New(sess)

	// Upload the image to S3
	_, err = s3Svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("failed putobject; %w", err)
	}

	return nil
}
