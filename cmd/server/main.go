package main

import (
	"chatapp/internal/config"
	"chatapp/internal/handler"
	"chatapp/internal/service"
	"chatapp/pkg/database"
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

	mediaService := service.NewMediaService(
		db,
		mediaStorage,
	)

	mediaHandler := handler.NewMediaHandler(
		mediaService,
	)

	router := gin.Default()

	// Setup routes
	handler.Setup(router, &handler.Handlers{
		Media: mediaHandler,
	})

	router.Run(":8080")
}
