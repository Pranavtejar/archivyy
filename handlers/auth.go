package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"archivyy/auth"
	"archivyy/db"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// formError replies with an error. htmx swaps the fragment into #response;
// a plain browser POST falls back to a redirect carrying the message.
func formError(c echo.Context, path, msg string) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, msg)
	}
	return c.Redirect(http.StatusSeeOther, path+"?error="+msg)
}

// redirect sends the browser to path. htmx ignores Location on a 303, so it
// needs HX-Redirect instead.
func redirect(c echo.Context, path string) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", path)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, path)
}

func Signup(c echo.Context) error {
	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	password := c.FormValue("password")
	confirm := c.FormValue("confirm_password")

	switch {
	case name == "" || email == "" || password == "":
		return formError(c, "/signup", "All fields are required")
	case len(password) < 8:
		return formError(c, "/signup", "Password must be at least 8 characters")
	case password != confirm:
		return formError(c, "/signup", "Passwords do not match")
	}

	exists, err := db.EmailExists(email)
	if err != nil {
		return formError(c, "/signup", "Something went wrong")
	}
	if exists {
		return formError(c, "/signup", "That email is already registered")
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		return formError(c, "/signup", "Something went wrong")
	}

	userID, err := db.CreateUser(uuid.NewString(), name, email, hashed)
	if err != nil {
		return formError(c, "/signup", "Something went wrong")
	}

	user, err := db.GetUserByEmail(email)
	if err != nil {
		return formError(c, "/signup", "Something went wrong")
	}

	token, err := auth.CreateJWT(user.UUID, name, userID)
	if err != nil {
		return formError(c, "/signup", "Something went wrong")
	}

	c.SetCookie(auth.CreateCookie(token))
	return redirect(c, "/")
}

func Login(c echo.Context) error {
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	password := c.FormValue("password")

	if email == "" || password == "" {
		return formError(c, "/login", "Email and password are required")
	}

	user, err := db.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Same message whether the email is unknown or the password is
			// wrong, so the response cannot be used to enumerate accounts.
			return formError(c, "/login", "Invalid email or password")
		}
		return formError(c, "/login", "Something went wrong")
	}

	if !auth.CheckPassword(user.Password, password) {
		return formError(c, "/login", "Invalid email or password")
	}

	token, err := auth.CreateJWT(user.UUID, user.Name, user.ID)
	if err != nil {
		return formError(c, "/login", "Something went wrong")
	}

	c.SetCookie(auth.CreateCookie(token))
	return redirect(c, "/")
}

func Logout(c echo.Context) error {
	c.SetCookie(auth.ClearCookie())
	return redirect(c, "/login")
}
