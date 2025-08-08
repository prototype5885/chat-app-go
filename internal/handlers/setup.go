package handlers

import (
	"chatapp-backend/internal/hub"
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var redisCtx = context.Background()
var db *sql.DB

var validate *validator.Validate

func Setup(isHttps bool, _redisClient *redis.Client, address string, port string, tlsCert string, tlsKey string, _sugar *zap.SugaredLogger, _db *sql.DB) error {
	sugar = _sugar
	redisClient = _redisClient
	db = _db

	validate = validator.New(validator.WithRequiredStructEnabled())

	http.HandleFunc("GET /api/test", Test)

	http.HandleFunc("POST /api/auth/login", Login)
	http.HandleFunc("POST /api/auth/register", Register)
	http.HandleFunc("GET /api/auth/newSession", Middleware(NewSession))

	http.HandleFunc("GET /api/isLoggedIn", Middleware(func(userID uint64, w http.ResponseWriter, r *http.Request) {}))

	http.HandleFunc("GET /api/user/fetch", Middleware(GetUserInfo))
	http.HandleFunc("POST /api/user/update", Middleware(UpdateUserInfo))

	http.HandleFunc("POST /api/server/create", Middleware(CreateServer))
	http.HandleFunc("GET /api/server/fetch", Middleware(SessionVerifier(GetServerList)))
	http.HandleFunc("POST /api/server/delete", Middleware(DeleteServer))
	http.HandleFunc("POST /api/server/rename", Middleware(RenameServer))

	http.HandleFunc("POST /api/channel/create", Middleware(CreateChannel))
	http.HandleFunc("GET /api/channel/fetch", Middleware(SessionVerifier(GetChannelList)))

	http.HandleFunc("POST /api/message/create", Middleware(CreateMessage))
	http.HandleFunc("GET /api/message/fetch", Middleware(SessionVerifier(GetMessageList)))
	http.HandleFunc("POST /api/message/delete", Middleware(DeleteMessage))

	http.HandleFunc("GET /api/members/fetch", Middleware(SessionVerifier(GetMemberList)))

	http.HandleFunc("GET /api/email/confirm", ConfirmEmail)

	http.Handle("/cdn/", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))

	http.HandleFunc("/ws", Middleware(hub.HandleClient))

	if isHttps {
		return http.ListenAndServeTLS(fmt.Sprintf("%s:%s", address, port), tlsCert, tlsKey, nil)
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%s", address, port), nil)
}
