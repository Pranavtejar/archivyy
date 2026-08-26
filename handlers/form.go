
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
)

var s3client *s3.Client

func Init() {
	admin := os.Getenv("S3_ACCESS_KEY")
	secret := os.Getenv("S3_SECRET_KEY")

	cfg := aws.Config{
		Region: "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(
			admin,
			secret,
			"",
		),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               "http://localhost:8333",
					HostnameImmutable: true,
				}, nil
			},
		),
	}

	s3client = s3.NewFromConfig(cfg)
}
//remove fmt error handling in the future 
func Upload(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		fmt.Println("MULTIPART ERROR:", err)

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	fmt.Println("VALUES:", form.Value)
	fmt.Println("FILES:", form.File)

	files := form.File["file"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "no file uploaded",
		})
	}

	file := files[0]

	fmt.Println("FIELD: file")
	fmt.Println("FILENAME:", file.Filename)
	fmt.Println("TYPE:", file.Header.Get("Content-Type"))

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to open file",
		})
	}
	defer src.Close()

	_, err = s3client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String("archive"),
		Key:         aws.String(file.Filename),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
	})
	if err != nil {
		fmt.Println("S3 UPLOAD ERROR:", err)

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "File uploaded",
		"file":    file.Filename,
	})
}

