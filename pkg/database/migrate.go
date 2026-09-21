package database

import (
	"log"

	"gorm.io/gorm"

	"chatapp/internal/model"
)

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&model.Account{},
		&model.User{},
		&model.DeviceSession{},
		&model.VerificationCode{},
		&model.Media{},
		&model.Post{},
		&model.PostLike{},
		&model.PostComment{},
		&model.Room{},
		&model.RoomMember{},
		&model.Message{},
		&model.MessageStatus{},
		&model.MessageReaction{},
	)
	if err != nil {
		log.Fatalf("Migration thất bại: %v", err)
	}

	log.Println("Migration hoàn tất")
}
