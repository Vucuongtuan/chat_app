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
		&model.QRLoginSession{},
		&model.SecondaryOTPSession{},
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
		&model.MessageHidden{},
		&model.MessageTarget{},
		&model.PinnedMessage{},
		&model.Poll{},
		&model.PollOption{},
		&model.PollVote{},
		&model.RoomSchedule{},
		&model.RoomActivity{},
		&model.ActivityParticipant{},
	)
	if err != nil {
		log.Fatalf("Migration thất bại: %v", err)
	}

	log.Println("Migration hoàn tất")
}
