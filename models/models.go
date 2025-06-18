package models

type User struct {
	ID       uint64
	UserName string
	Email    string
	Password []byte
}

type Server struct {
	ID      uint64 `json:"id,string"`
	OwnerID uint64 `json:"ownerID,string"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Banner  string `json:"banner"`
}

type Channel struct {
	ID       uint64 `json:"id,string"`
	ServerID uint64 `json:"serverID,string"`
	Name     string `json:"name"`
}

type Message struct {
	ID          uint64 `json:"id,string"`
	ChannelID   uint64 `json:"channelID,string"`
	UserID      uint64 `json:"userID,string"`
	Message     string `json:"message"`
	Attachments []byte `json:"attachments"`
	Edited      bool   `json:"edited"`
}
