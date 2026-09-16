package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	Content string `json:"content" gorm:"type:text;not null"`

	UserId uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	Likes    []PostLike    `json:"likes,omitempty" gorm:"foreignKey:PostId"`
	Comments []PostComment `json:"comments,omitempty" gorm:"foreignKey:PostId"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PostLike struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	PostId uuid.UUID `json:"post_id" gorm:"type:char(36);not null;index:idx_post_user_like,unique"`
	UserId uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index:idx_post_user_like,unique"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (pl *PostLike) BeforeCreate(tx *gorm.DB) (err error) {
	if pl.ID == uuid.Nil {
		pl.ID = uuid.New()
	}
	return nil
}

type PostComment struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`

	PostId  uuid.UUID `json:"post_id" gorm:"type:char(36);not null;index"`
	UserId  uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Content string    `json:"content" gorm:"type:text;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (pc *PostComment) BeforeCreate(tx *gorm.DB) (err error) {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	return nil
}
