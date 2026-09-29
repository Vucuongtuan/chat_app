package main

import (
	"log"

	"chatapp/internal/config"
	"chatapp/internal/handler"
	"chatapp/internal/middleware"
	"chatapp/internal/repository"
	"chatapp/internal/service"
	"chatapp/pkg/database"
	"chatapp/pkg/i18n"
	"chatapp/pkg/mailer"
	redisclient "chatapp/pkg/redis"
	"chatapp/pkg/storage"

	"chatapp/pkg/eventbus"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	redis, err := redisclient.New(cfg)
	if err != nil {
		log.Fatalf("No connect Redis: %v", err)
	}
	defer redis.Close()

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

	// Auth & Devices & 2FA
	authRepo := repository.NewAuthRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	mailService := mailer.NewMailer(cfg)

	// in-process event bus for async events (websocket, notifications)
	bus := eventbus.New()

	authService := service.NewAuthService(authRepo, deviceRepo, mailService, cfg, bus)
	authHandler := handler.NewAuthHandler(authService)

	// Room, Message, Post, Room Events, Room Activities
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	postRepo := repository.NewPostRepository(db)
	roomEventRepo := repository.NewRoomEventRepository(db)
	roomActivityRepo := repository.NewRoomActivityRepository(db)

	roomService := service.NewRoomService(roomRepo, userRepo, bus)
	messageService := service.NewMessageService(messageRepo, roomRepo, bus)
	postService := service.NewPostService(postRepo, userRepo)
	roomEventService := service.NewRoomEventService(roomEventRepo, roomRepo, messageRepo, bus)
	roomActivityService := service.NewRoomActivityService(roomActivityRepo, roomRepo, messageRepo, bus)

	roomHandler := handler.NewRoomHandler(roomService)
	messageHandler := handler.NewMessageHandler(messageService)
	postHandler := handler.NewPostHandler(postService)
	roomEventHandler := handler.NewRoomEventHandler(roomEventService)
	roomActivityHandler := handler.NewRoomActivityHandler(roomActivityService)

	// Middleware
	authMiddleware := middleware.Auth(cfg.JWTSecret)

	router := gin.Default()
	router.Use(i18n.Middleware())

	// Setup routes
	handler.Setup(router, &handler.Handlers{
		Auth:         authHandler,
		Media:        mediaHandler,
		User:         userHandler,
		Room:         roomHandler,
		Message:      messageHandler,
		Post:         postHandler,
		RoomEvent:    roomEventHandler,
		RoomActivity: roomActivityHandler,
	}, authMiddleware)

	router.Run(":8080")
}
