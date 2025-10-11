package models

type User struct {
	ID          int64  `json:"id,string,omitempty"`
	Email       string `json:"email,omitempty"`
	UserName    string `json:"userName,omitempty"`
	DisplayName string `json:"displayName"`
	Picture     string `json:"picture"`
	Password    []byte `json:"password,omitempty"`
}

type Server struct {
	ID      int64  `json:"id,string"`
	OwnerID int64  `json:"ownerID,string"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Banner  string `json:"banner"`
}

type Channel struct {
	ID       int64  `json:"id,string"`
	ServerID int64  `json:"serverID,string"`
	Name     string `json:"name"`
}

type Message struct {
	ID          int64  `json:"id,string"`
	ChannelID   int64  `json:"channelID,string"`
	UserID      int64  `json:"userID,string"`
	Message     string `json:"message"`
	Attachments string `json:"attachments"`
	Edited      bool   `json:"edited"`
	User        User   `json:"user"`
}

type ConfigFile struct {
	Address           string
	Port              string
	BehindNginx       bool
	TlsCert           string
	TlsKey            string
	Cors              bool
	PrintHttpRequests bool
	LogToFile         bool
	LogLevel          string
	JwtSecret         string
	SnowflakeWorkerID int64
	SelfContained     bool
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
