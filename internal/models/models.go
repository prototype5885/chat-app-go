package models

type User struct {
	ID          uint64 `msgpack:"id,omitempty"`
	Email       string `msgpack:"email,omitempty"`
	UserName    string `msgpack:"userName,omitempty"`
	DisplayName string `msgpack:"displayName"`
	Picture     string `msgpack:"picture"`
	Password    []byte `msgpack:"password,omitempty"`
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
	User        User   `msgpack:"user"`
}

type ConfigFile struct {
	Address           string
	Port              string
	TlsCert           string
	TlsKey            string
	PrintHttpRequests bool
	JwtSecret         string
	SnowflakeWorkerID uint64
	DbUser            string
	DbPassword        string
	DbAddress         string
	DbPort            string
	DbDatabase        string
	SmtpUsername      string
	SmtpPassword      string
	SmtpServer        string
	SmtpPort          int
}
