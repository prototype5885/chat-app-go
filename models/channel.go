package models

type Channel struct {
	ID       uint64 `gorm:"autoIncrement:false"`
	ServerID uint64
	Name     string
}
