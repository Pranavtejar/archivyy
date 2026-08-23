package auth

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// tokenTTL is the lifetime of both the JWT and the cookie carrying it.
// These must agree, or the browser keeps sending a cookie the server rejects.
const tokenTTL = 24 * time.Hour

func jwtSecret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET is not set")
	}
	return []byte(s), nil
}

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func CreateJWT(uuid string, name string, userID int64) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"uuid":    uuid,
		"name":    name,
		"user_id": userID,
		"exp":     time.Now().Add(tokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseJWT(tokenString string) (jwt.MapClaims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func CreateCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     "auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(tokenTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearCookie expires the auth cookie. MaxAge < 0 deletes it immediately.
func ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
}

// Middleware requires a valid session, redirecting to /login otherwise.
func Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !loadUser(c) {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		return next(c)
	}
}

// Optional populates user context when signed in, but allows anonymous
// requests through. Used for public pages that change when logged in.
func Optional(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		loadUser(c)
		return next(c)
	}
}

// loadUser validates the auth cookie and sets user_id, name and uuid on the
// context. Reports whether a valid session was found.
func loadUser(c echo.Context) bool {
	cookie, err := c.Cookie("auth")
	if err != nil {
		return false
	}

	claims, err := ParseJWT(cookie.Value)
	if err != nil {
		return false
	}

	// encoding/json decodes every JSON number into float64, so the claim must
	// be asserted as float64 before narrowing to int64.
	floatID, ok := claims["user_id"].(float64)
	if !ok {
		return false
	}

	name, _ := claims["name"].(string)
	uuid, _ := claims["uuid"].(string)

	c.Set("user_id", int64(floatID))
	c.Set("name", name)
	c.Set("uuid", uuid)
	return true
}
