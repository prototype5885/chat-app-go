package models

type User struct {
	ID          uint64 `msgpack:"id"`
	Email       string `msgpack:"email"`
	UserName    string `msgpack:"userName"`
	DisplayName string `msgpack:"displayName"`
	Picture     string `msgpack:"picture"`
	Password    []byte `msgpack:"password"`
}

type Server struct {
	ID      uint64 `msgpack:"id"`
	OwnerID uint64 `msgpack:"ownerID"`
	Name    string `msgpack:"name"`
	Picture string `msgpack:"picture"`
	Banner  string `msgpack:"banner"`
}

type Channel struct {
	ID       uint64 `msgpack:"id"`
	ServerID uint64 `msgpack:"serverID"`
	Name     string `msgpack:"name"`
}

type Message struct {
	ID          uint64 `msgpack:"id"`
	ChannelID   uint64 `msgpack:"channelID"`
	UserID      uint64 `msgpack:"userID"`
	Message     string `msgpack:"message"`
	Attachments []byte `msgpack:"attachments"`
	Edited      bool   `msgpack:"edited"`
}
