package email

import (
	"fmt"
	"net/smtp"
	"net/url"

	"github.com/redis/go-redis/v9"
)

// var redisClient *redis.Client
// var redisCtx = context.Background()

var server string
var address string
var username string
var password string
var fullServerAddress string

func Setup(_redisClient *redis.Client, _server string, _port int, _username string, _password string, _fullServerAddress string) {
	// redisClient = _redisClient
	server = _server
	address = fmt.Sprintf("%s:%d", _server, _port)
	username = _username
	password = _password
	fullServerAddress = _fullServerAddress
}

func sendEmail(email []string, subject string, message string) error {
	auth := smtp.PlainAuth("", username, password, server)

	msg := fmt.Appendf(nil, "To: %s\r\n", email[0])
	msg = fmt.Append(msg, "MIME-version: 1.0;\r\n")
	msg = fmt.Append(msg, "Content-Type: text/html; charset=\"UTF-8\";\r\n")
	msg = fmt.Appendf(msg, "Subject: %s\r\n", subject)
	msg = fmt.Append(msg, "\r\n")
	msg = fmt.Appendf(msg, "%s\r\n", message)

	return smtp.SendMail(address, auth, username, email, msg)
}

func SendEmailConfirmation(email string, username string, token string) error {
	link := fmt.Sprintf("%s/api/email/confirm?token=%s", fullServerAddress, url.QueryEscape(token))

	subject := "Email confirmation"
	message := fmt.Sprintf(`
	<html>
		<body>
			<h2>Hallo %s!</h2>
			<a href="%s">Confirm your email by clicking here</a>
		</body>
	</html>`,
		username, link)

	return sendEmail([]string{email}, subject, message)
}
