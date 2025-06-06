package models

type User struct {
	ID       uint64 `gorm:"autoIncrement:false"`
	UserName string `gorm:"not null; unique; size:16"`
	Email    string `gorm:"not null; unique; size:64"`
	Password []byte `gorm:"not null; size:60"`
	// Messages Message `gorm:"constraint:OnDelete:CASCADE;"`
}
