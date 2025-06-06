package models

type Message struct {
	ID        uint64 `gorm:"autoIncrement:false"`
	ChannelID uint64
	Message   string
	Edited    bool
}
