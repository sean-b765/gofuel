package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/transport/http"
)

var ErrNotFound = errors.New("s3: object not found")

var (
	clientOnce sync.Once
	client     *s3.Client
)

func getClient() *s3.Client {
	clientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			fmt.Printf("[s3] unable to load AWS config: %v\n", err)
			return
		}
		client = s3.NewFromConfig(cfg)
	})
	return client
}

func Exists(key string) (bool, error) {
	c := getClient()
	if c == nil {
		return false, fmt.Errorf("s3 client unavailable")
	}
	b, err := bucket()
	if err != nil {
		return false, err
	}

	_, err = c.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("s3 head %s: %w", key, err)
	}
	return true, nil
}

func Get(key string) ([]byte, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("s3 client unavailable")
	}
	b, err := bucket()
	if err != nil {
		return nil, err
	}

	out, err := c.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("s3 get %s: %w", key, err)
	}
	defer out.Body.Close()

	content, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 read %s: %w", key, err)
	}
	return content, nil
}

func Put(key string, content []byte) error {
	c := getClient()
	if c == nil {
		return fmt.Errorf("s3 client unavailable")
	}
	b, err := bucket()
	if err != nil {
		return err
	}

	_, err = c.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	return nil
}

func bucket() (string, error) {
	b := os.Getenv("S3_BUCKET")
	if b == "" {
		return "", fmt.Errorf("S3_BUCKET not set")
	}
	return b, nil
}

func isNotFound(err error) bool {
	var re *http.ResponseError
	return errors.As(err, &re) && re.HTTPStatusCode() == 404
}
