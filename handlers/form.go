// the pigeons have taken over the server room

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
)

type FileMeta struct {
	Key         string `json:"key"`
	Title       string `json:"title,omitempty"`
	ContentType string `json:"ContentType"`
	Thumb       string `json:"thumb,omitempty"`
	Views       int    `json:"views"`
	ViewedBy 	 []string `json:"viewedBy,omitempty"`
	Tags 			[]string `json:"tags,omitempty"`
}

type Meta struct {
	files []FileMeta
	sync.RWMutex
}

func (meta *Meta) write(data FileMeta) error {
	meta.Lock()
	defer meta.Unlock()

	meta.files = append(meta.files, data)

	b, err := json.MarshalIndent(meta.files, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile("meta.json", b, 0644)
}

func (meta *Meta) incrementViews(key, name string) error {
	meta.Lock()
	defer meta.Unlock()
	fmt.Printf("VIEW DEBUG: key=%s name=%q\n", key, name)
	
	for i := range meta.files {
		if meta.files[i].Key != key {
			continue
		}
		for _, viewedBy := range meta.files[i].ViewedBy {
			if viewedBy == name {
				return nil // User has already viewed this file, no need to increment views
			}
		}
		meta.files[i].Views++
		meta.files[i].ViewedBy = append(meta.files[i].ViewedBy, name)
		break	
	}

	b, err := json.MarshalIndent(meta.files, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile("meta.json", b, 0644)
}

var s3client *s3.Client
var metaData *Meta

func Init() {
	metaData = &Meta{
		files: make([]FileMeta, 0),
	}

	f, err := os.ReadFile("meta.json")

	if err != nil && !os.IsNotExist(err) {
		fmt.Println("META READ ERROR:", err)
	}

	if len(f) > 0 {
		if err := json.Unmarshal(f, &metaData.files); err != nil {
			fmt.Println("META JSON ERROR:", err)
			metaData.files = make([]FileMeta, 0)
		}
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

func Upload(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
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

	thumbName, err := saveThumbnail(form)
	if err != nil {
		fmt.Println("THUMBNAIL ERROR:", err)
	}

	newMeta := FileMeta{
		Key:         file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		Views:       0,
	}

	if title != "" {
		newMeta.Title = title
	}

	if thumbName != "" {
		newMeta.Thumb = thumbName
	}

	if err := metaData.write(newMeta); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to save metadata",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "File uploaded",
		"file":    file.Filename,
	})
}

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

func Display(c echo.Context) error {
	metaData.RLock()

	items := make([]FileMeta, len(metaData.files))
	copy(items, metaData.files)

	metaData.RUnlock()

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Views > items[j].Views
	})

	type item struct {
		Key         string `json:"key"`
		Title       string `json:"title"`
		ContentType string `json:"ContentType"`
		Preview     string `json:"preview"`
		Views       int    `json:"views"`
	}

	result := make([]item, 0, len(items))

	for _, obj := range items {
		title := obj.Title

		if title == "" {
			title = obj.Key
		}

		preview := ""

		if obj.Thumb != "" {
			preview = "/static/images/" + obj.Thumb
		}

		result = append(result, item{
			Key:         obj.Key,
			Title:       title,
			ContentType: obj.ContentType,
			Preview:     preview,
			Views:       obj.Views,
		})
	}

	return c.JSON(http.StatusOK, result)
}

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
	name, _ := c.Get("name").(string)
	if err := metaData.incrementViews(key, name); err != nil {
		fmt.Println("VIEW COUNT ERROR:", err)
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

func Search(c echo.Context) error {
	query := strings.ToLower(c.QueryParam("q"))

	metaData.RLock()
	defer metaData.RUnlock()

	var results []FileMeta

	for _, file := range metaData.files {
		if strings.Contains(strings.ToLower(file.Key), query) ||
			strings.Contains(strings.ToLower(file.Title), query) {
			results = append(results, file)
		}
	}

	return c.JSON(http.StatusOK, results)
}
