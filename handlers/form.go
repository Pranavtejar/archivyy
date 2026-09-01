// papa

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
)

type Meta struct {
	data map[string]string
	order []string
	sync.RWMutex
}

func (meta *Meta) write(data map[string]string) {
	meta.Lock()
	defer meta.Unlock()

	var all []map[string]string

	f, err := os.ReadFile("meta.json")
	if err == nil && len(f) > 0 {
		json.Unmarshal(f, &all)
	}

	all = append(all, data)

	b, _ := json.MarshalIndent(all, "", " ")
	os.WriteFile("meta.json", b, 0644)

	if len(meta.order) > 15 {
		meta.order = meta.order[1:]
	}
}

var s3client *s3.Client
var metaData *Meta

func Init() {
	metaData = &Meta{
		data:  make(map[string]string),
		order: []string{},
	}

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
					URL:               "http://[2a01:4f9:3a:276e::1738]:8333",
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

	title := strings.TrimSpace(c.FormValue("title"))

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to open file",
		})
	}

	data, err := io.ReadAll(src)
	src.Close()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read file",
		})
	}

	_, err = s3client.PutObject(c.Request().Context(), &s3.PutObjectInput{
		Bucket:      aws.String("archive"),
		Key:         aws.String(file.Filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(file.Header.Get("Content-Type")),
	})
	if err != nil {
		fmt.Println("S3 UPLOAD ERROR:", err)

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	thumbName, err := saveThumbnail(form)
	if err != nil {
		fmt.Println("THUMBNAIL ERROR:", err)
	}

	newMeta := map[string]string{
		"key":         file.Filename,
		"ContentType": file.Header.Get("Content-Type"),
	}

	if title != "" {
		newMeta["title"] = title
	}

	if thumbName != "" {
		newMeta["thumb"] = thumbName
	}

	metaData.write(newMeta)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "File uploaded",
		"file":    file.Filename,
	})
}

// saveThumbnail stores an optional user-selected thumbnail from the upload
// form into static/images. It returns the saved filename (empty if none was
// provided). Thumbnails are chosen by the user; nothing is generated here.
func saveThumbnail(form *multipart.Form) (string, error) {
	thumbs := form.File["thumbnail"]
	if len(thumbs) == 0 {
		return "", nil
	}

	t := thumbs[0]

	src, err := t.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dir := "static/images"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	name := slugifyKey(t.Filename)
	if name == "" {
		name = "thumb.jpg"
	}

	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return name, nil
}

// Display returns the metadata for every stored object as JSON. It performs
// no S3 access or file download; each entry carries a preview URL that points
// to a lightweight thumbnail under /static/images for the frontend cards.
func Display(c echo.Context) error {
	var all []map[string]string

	f, err := os.ReadFile("meta.json")
	if err == nil && len(f) > 0 {
		json.Unmarshal(f, &all)
	}

	type item struct {
		Key         string `json:"key"`
		Title       string `json:"title"`
		ContentType string `json:"ContentType"`
		Preview     string `json:"preview"`
	}

	items := make([]item, 0, len(all))

	for _, obj := range all {
		key := obj["key"]

		contentType := obj["ContentType"]
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		title := obj["title"]
		if title == "" {
			title = key
		}

		preview := ""
		if thumb := obj["thumb"]; thumb != "" {
			preview = "/static/images/" + thumb
		}

		items = append(items, item{
			Key:         key,
			Title:       title,
			ContentType: contentType,
			Preview:     preview,
		})
	}

	return c.JSON(http.StatusOK, items)
}

// slugifyKey turns a stored name into a URL-safe filename for its thumbnail.
func slugifyKey(key string) string {
	var b strings.Builder

	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)

		default:
			b.WriteByte('-')
		}
	}

	return b.String()
}

// sanitizeFilename scrubs characters that would break out of a quoted MIME
// header.
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

// Stream serves a single object from the archive bucket.
// It supports HTTP Range requests.
func Stream(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("filename")

	if key == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String("archive"),
		Key:    aws.String(key),
	}

	if rh := c.Request().Header.Get("Range"); rh != "" {
		input.Range = aws.String(rh)
	}

	res, err := s3client.GetObject(ctx, input)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	defer res.Body.Close()

	h := c.Response().Header()

	contentType := "application/octet-stream"

	if res.ContentType != nil && *res.ContentType != "" {
		contentType = *res.ContentType
	}

	h.Set("Content-Type", contentType)
	h.Set("Accept-Ranges", "bytes")
	h.Set(
		"Content-Disposition",
		fmt.Sprintf(
			`inline; filename="%s"`,
			sanitizeFilename(key),
		),
	)

	status := http.StatusOK

	if res.ContentRange != nil && *res.ContentRange != "" {
		h.Set("Content-Range", *res.ContentRange)
		status = http.StatusPartialContent
	}

	if res.ContentLength != nil {
		h.Set(
			"Content-Length",
			strconv.FormatInt(*res.ContentLength, 10),
		)
	}

	return c.Stream(
		status,
		contentType,
		res.Body,
	)
}
