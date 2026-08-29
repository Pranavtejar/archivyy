package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"mime/multipart"
	"net/textproto"

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

// remove fmt error handling in the future
// create new func to retrive all the uploaded data and sort it based of a recomendation algorithm
func Upload(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		fmt.Println("MULTIPART ERROR:", err)

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	files := form.File["file"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "no file uploaded",
		})
	}

	file := files[0]

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to open file",
		})
	}
	defer src.Close()

	_, err = s3client.PutObject(c.Request().Context(), &s3.PutObjectInput{
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

// Display streams every object in the "archive" bucket back as a
// multipart/mixed response, one part per object. The part headers carry the
// stored content type and the object key as the filename so a client can
// rebuild each file without knowing what is in the archive ahead of time.
func Display(c echo.Context) error {
	ctx := c.Request().Context()

	paginator := s3.NewListObjectsV2Paginator(
		s3client,
		&s3.ListObjectsV2Input{
			Bucket:  aws.String("archive"),
			MaxKeys: aws.Int32(15),
		},
	)

	// Gather every key before writing the response head. Listing happens
	// before anything is flushed, so a failure here can still return a clean
	// error instead of corrupting an already-started multipart body.
	var keys []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to list the archive",
			})
		}
		for _, object := range page.Contents {
			if object.Key != nil && *object.Key != "" {
				keys = append(keys, *object.Key)
			}
		}
	}

	mw := multipart.NewWriter(c.Response())
	c.Response().Header().Set(
		"Content-Type",
		"multipart/mixed; boundary="+mw.Boundary(),
	)

	for _, key := range keys {
		res, err := s3client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String("archive"),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("display: skipping %q: %v", key, err)
			continue
		}

		contentType := "application/octet-stream"
		if res.ContentType != nil && *res.ContentType != "" {
			contentType = *res.ContentType
		}

		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", contentType)
		header.Set(
			"Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(key)),
		)

		part, err := mw.CreatePart(header)
		if err != nil {
			res.Body.Close()
			log.Printf("display: skipping %q: %v", key, err)
			continue
		}

		_, err = io.Copy(part, res.Body)
		res.Body.Close()
		if err != nil {
			log.Printf("display: truncated %q: %v", key, err)
			continue
		}
	}

	return mw.Close()
}

// sanitizeFilename scrubs characters that would break out of a quoted MIME
// header (CRLF, quotes and backslashes) in the Content-Disposition filename.
func sanitizeFilename(key string) string {
	var b strings.Builder
	for _, r := range key {
		switch r {
		case '\r', '\n', '"', '\\':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
