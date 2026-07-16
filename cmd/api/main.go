package main

import (
	"context"
	"log"
	"os"
	"time"
	"warehouse-api/internal/auth"
	"warehouse-api/internal/middleware"
	"warehouse-api/internal/user"
	"warehouse-api/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.NewConnection(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Succesfully connected to database")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET env var is required")
	}
	tokenService := auth.NewTokenService(jwtSecret, 24*time.Hour)

	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo)
	UserHandler := user.NewUserHandler(userService, tokenService)

	r := gin.Default()

	r.Use(middleware.Timeout(5 * time.Second))

	registerUserRoutes(r, UserHandler, tokenService)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerUserRoutes(r *gin.Engine, h *user.Handler, tokenService *auth.TokenService) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	users := r.Group("/users")
	users.Use(middleware.RequiredAuth(tokenService))
	{
		users.GET("", h.GetAll)
		users.GET("/:id", h.GetByID)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", middleware.RequiredRole("admin"), h.Delete)
	}
}
