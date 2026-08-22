package auth

import (
	"time"
	"os"

	"youtubevid/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"	
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func CreateJWT(uuid string, name string, userID int64) (string, error) {
	claims := jwt.MapClaims{
		"uuid": uuid,
		"name": name,
		"user_id": userID, 
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ParseJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, echo.NewHTTPError(401, "Unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, echo.NewHTTPError(401, "Invalid token")
	}
}

func CreateCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name: "auth",
		Value: token,
		Path: "/",
		HttpOnly: true,
		MaxAge: 60*60*24*7,
		SameSite: http.SameSiteLaxMode,
	}
}

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("auth")
		if err != nil {
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		claims, ok := ParseJWT(cookie.Value)
		if !ok {
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		// JWT unmarshals JSON numbers as float64. 
		// We must assert to float64 first, then convert to int64 to avoid a panic.
		floatID, ok := claims["user_id"].(float64)
		if !ok {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		userID := int64(floatID)
	
		uuid, _ := claims["UUID"].(string)
		name := claims["name"].(string)

		c.Set("user_id", userID)
		c.Set("name", name)
		c.Set("UUID", uuid)
		return next(c)
	}
}

