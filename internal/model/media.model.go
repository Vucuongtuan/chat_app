package model

import "gorm.io/gorm"

type Media struct {
	gorm.Model

	Url  string `json:"url" gorm:"type:text;not null"`
	Type string `json:"type" gorm:"type:varchar(20);not null"` // "image", "video"

	OwnerType string `json:"owner_type" gorm:"type:varchar(30);not null;index"` // "user_avatar", "user_cover", "post", "message"
	OwnerId   uint   `json:"owner_id" gorm:"type:bigint;not null;index"`

	SortOrder int `json:"sort_order" gorm:"type:int;default:0"` // thứ tự ảnh khi post có nhiều ảnh
}
