package model

import "gorm.io/gorm"

type Post struct {
	gorm.Model

	Content string `json:"content" gorm:"type:text;not null"`

	UserId uint `json:"user_id" gorm:"type:bigint;not null;index"`

	//
	Likes    []PostLike    `json:"likes,omitempty" gorm:"foreignKey:PostId"`
	Comments []PostComment `json:"comments,omitempty" gorm:"foreignKey:PostId"`
}

type PostLike struct {
	gorm.Model

	PostId uint `json:"post_id" gorm:"type:bigint;not null;index:idx_post_user_like,unique"`
	UserId uint `json:"user_id" gorm:"type:bigint;not null;index:idx_post_user_like,unique"`
}

type PostComment struct {
	gorm.Model

	PostId  uint   `json:"post_id" gorm:"type:bigint;not null;index"`
	UserId  uint   `json:"user_id" gorm:"type:bigint;not null;index"`
	Content string `json:"content" gorm:"type:text;not null"`
}
