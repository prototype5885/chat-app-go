package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/models"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

var sugar *zap.SugaredLogger
var db *sql.DB

var validate *validator.Validate

func Setup(isHttps bool, cfg *models.ConfigFile, _sugar *zap.SugaredLogger, _db *sql.DB) error {
	sugar = _sugar
	db = _db

	validate = validator.New(validator.WithRequiredStructEnabled())

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/test", Test)

	mux.HandleFunc("POST /api/auth/login", Login)
	mux.HandleFunc("POST /api/auth/register", Register)
	mux.HandleFunc("GET /api/auth/newSession", Middleware(NewSession))
	mux.HandleFunc("GET /api/auth/isLoggedIn", Middleware(func(userID uint64, w http.ResponseWriter, r *http.Request) {}))

	mux.HandleFunc("GET /api/user/fetch", Middleware(GetUserInfo))
	mux.HandleFunc("POST /api/user/update", Middleware(UpdateUserInfo))

	mux.HandleFunc("POST /api/server/create", Middleware(CreateServer))
	mux.HandleFunc("GET /api/server/fetch", Middleware(SessionVerifier(GetServerList)))
	mux.HandleFunc("POST /api/server/delete", Middleware(DeleteServer))
	mux.HandleFunc("POST /api/server/rename", Middleware(RenameServer))

	mux.HandleFunc("POST /api/channel/create", Middleware(CreateChannel))
	mux.HandleFunc("GET /api/channel/fetch", Middleware(SessionVerifier(GetChannelList)))

	mux.HandleFunc("POST /api/message/create", Middleware(CreateMessage))
	mux.HandleFunc("GET /api/message/fetch", Middleware(SessionVerifier(GetMessageList)))
	mux.HandleFunc("POST /api/message/delete", Middleware(DeleteMessage))

	mux.HandleFunc("GET /api/members/fetch", Middleware(SessionVerifier(GetMemberList)))

	mux.HandleFunc("GET /api/email/confirm", ConfirmEmail)

	mux.Handle("/cdn/", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))

	mux.HandleFunc("/ws", Middleware(hub.HandleClient))

	var handler http.Handler
	if cfg.PrintHttpRequests {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			mux.ServeHTTP(w, r)
			duration := time.Since(start)

			fmt.Printf("[%s] %s %s from %s - Duration: %v\n", start.Format(time.RFC1123), r.Method, r.URL, r.RemoteAddr, duration)
		})
	} else {
		handler = mux
	}

	address := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)

	if isHttps {
		return http.ListenAndServeTLS(address, cfg.TlsCert, cfg.TlsKey, handler)
	}
	return http.ListenAndServe(address, handler)
}
