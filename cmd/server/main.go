package main

import (
	"chatapp/internal/config"
	"chatapp/internal/handler"
	"chatapp/internal/middleware"
	"chatapp/internal/repository"
	"chatapp/internal/service"
	"chatapp/pkg/database"
	"chatapp/pkg/i18n"
	"chatapp/pkg/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// database & migration
	db := database.Connect(cfg)
	database.Migrate(db)

	// storeage
	mediaStorage := storage.NewLocalStorage(
		"./uploads",
		"/uploads",
	)

	mediaRepo := repository.NewMediaRepository(db)
	mediaService := service.NewMediaService(
		mediaRepo,
		mediaStorage,
	)

	mediaHandler := handler.NewMediaHandler(
		mediaService,
	)

	// User
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	// Middleware
	authMiddleware := middleware.Auth(cfg.JWTSecret)

	router := gin.Default()
	router.Use(i18n.Middleware())

	// Setup routes
	handler.Setup(router, &handler.Handlers{
		Media: mediaHandler,
		User:  userHandler,
	}, authMiddleware)

	router.Run(":8080")
}
