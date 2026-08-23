package main

import (
	"log"
	"os"

	"archivyy/auth"
	"archivyy/db"
	"archivyy/handlers"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("reading .env: %v", err)
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is not set (see .env.example)")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./archivyy.db"
	}
	db.Init(dbPath)
	defer db.DB.Close()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())
	e.Renderer = handlers.NewRenderer("templates/*.html")
	e.Static("/static", "static")

	// Public. Optional resolves the session so pages can show who is signed in.
	e.GET("/", handlers.Home, auth.Optional)
	e.GET("/login", handlers.LoginPage, auth.Optional)
	e.GET("/signup", handlers.SignupPage, auth.Optional)
	e.POST("/login", handlers.Login)
	e.POST("/signup", handlers.Signup)
	e.POST("/logout", handlers.Logout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
