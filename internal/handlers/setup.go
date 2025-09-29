package handlers

import (
	"chatapp-backend/internal/models"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var db *sql.DB

func Setup(isHttps bool, cfg *models.ConfigFile, _sugar *zap.SugaredLogger, _db *sql.DB) error {
	sugar = _sugar
	db = _db

	r := chi.NewRouter()
	// r.Use(AllowCors)
	// mux.Use(middleware.RequestID)
	// mux.Use(middleware.RealIP)
	if cfg.PrintHttpRequests {
		r.Use(middleware.Logger)
	}

	r.Use(middleware.Recoverer)
	// mux.Use(middleware.Compress(5))

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(api chi.Router) {
		api.Get("/test", Test)
		api.Post("/testSqlite", TestSqlite)

		api.Route("/auth", func(r chi.Router) {
			r.Post("/login", Login)
			r.Post("/register", Register)
			r.With(UserVerifier).Get("/newSession", NewSession)
			r.With(UserVerifier).Get("/isLoggedIn", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		})

		api.Route("/user", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Get("/fetch", GetUserInfo)
			r.Post("/update", UpdateUserInfo)
		})

		api.Route("/server", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Post("/create", CreateServer)
			r.With(SessionVerifier).Get("/fetch", GetServerList)
			r.Post("/delete", DeleteServer)
			r.Post("/rename", RenameServer)
		})

		api.Route("/channel", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Post("/create", CreateChannel)
			r.With(SessionVerifier).Get("/fetch", GetChannelList)
		})

		api.Route("/message", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Post("/create", CreateMessage)
			r.With(SessionVerifier).Get("/fetch", GetMessageList)
			r.Post("/delete", DeleteMessage)
		})

		api.Route("/members", func(r chi.Router) {
			r.Use(UserVerifier)
			r.With(SessionVerifier).Get("/fetch", GetMemberList)
		})

		api.Route("/email", func(r chi.Router) {
			r.Get("/confirm", ConfirmEmail)
		})
	})

	var websocketPath string

	if cfg.BehindNginx {
		websocketPath = "/ws/"
	} else {
		websocketPath = "/ws"
		r.Handle("/cdn/*", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))
		r.Handle("/*", http.FileServer(http.Dir("./public/static")))
	}

	r.With(UserVerifier).Get(websocketPath, HandleWebSocket)

	address := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)

	if isHttps {
		return http.ListenAndServeTLS(address, cfg.TlsCert, cfg.TlsKey, r)
	}
	return http.ListenAndServe(address, r)
}
