package model

import "gorm.io/gorm"

type Account struct {
	gorm.Model

	Phone        string  `json:"phone" gorm:"type:varchar(15);not null;unique"`
	Mail         *string `json:"mail,omitempty" gorm:"type:varchar(100);unique"`
	PasswordHash string  `json:"-" gorm:"type:varchar(255);not null"`
	HashAlgo     string  `json:"-" gorm:"type:varchar(20);not null;default:'bcrypt'"`
	IsVerified   bool    `json:"is_verified" gorm:"type:boolean;not null;default:false"`

	// Foreign key to User Table
	User User `json:"user" gorm:"foreignKey:AccountId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
