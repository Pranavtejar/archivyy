package handlers

import (
	"html/template"
	"io"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(pattern string) *Renderer {
	return &Renderer{templates: template.Must(template.ParseGlob(pattern))}
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

// pageData carries the signed-in user (if any) to every template.
func pageData(c echo.Context) map[string]interface{} {
	name, _ := c.Get("name").(string)
	return map[string]interface{}{
		"LoggedIn": name != "",
		"Name":     name,
		"Error":    c.QueryParam("error"),
	}
}

func LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", pageData(c))
}

func SignupPage(c echo.Context) error {
	return c.Render(http.StatusOK, "signup.html", pageData(c))
}

func Home(c echo.Context) error {
	return c.Render(http.StatusOK, "home.html", pageData(c))
}

func ViewPage(c echo.Context) error {
	filename := c.Param("filename")
	if filename == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	data := pageData(c)
	data["Filename"] = filename
	data["StreamURL"] = "/stream/" + url.PathEscape(filename)
	return c.Render(http.StatusOK, "view.html", data)
}
