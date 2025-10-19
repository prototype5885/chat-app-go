package handlers

import (
	"chatapp-backend/internal/models"
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var db *sql.DB
var isHttps bool
var snowflakeNode *snowflake.Node

func Setup(_isHttps bool, cfg *models.ConfigFile, _sugar *zap.SugaredLogger, _db *sql.DB, _snowflakeNode *snowflake.Node) error {
	isHttps = _isHttps
	sugar = _sugar
	db = _db
	snowflakeNode = _snowflakeNode

	// this fixes problem serving flutter web wasm,
	// as by default it sends .mjs as text/plain
	err := mime.AddExtensionType(".mjs", "application/javascript")
	if err != nil {
		return err
	}

	r := chi.NewRouter()

	if cfg.RateLimiting {
		r.Use(httprate.LimitByIP(100, time.Minute))
	}

	if cfg.Cors {
		r.Use(AllowCors)
	}

	// mux.Use(middleware.RequestID)
	// mux.Use(middleware.RealIP)
	if cfg.PrintHttpRequests {
		r.Use(middleware.Logger)
	}

	r.Use(middleware.Recoverer)
	//r.Use(middleware.Compress(5))

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(api chi.Router) {
		api.Get("/test", Test)

		api.Route("/auth", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httprate.LimitByIP(5, time.Minute))
				r.Post("/login", Login)
				r.Post("/register", Register)
			})
			r.With(UserVerifier).Get("/newSession", NewSession)
			r.With(UserVerifier).Get("/isLoggedIn", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		})

		api.Route("/user", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Group(func(r chi.Router) {
				r.Use(httprate.LimitByIP(10, time.Minute))
				r.Post("/update", UpdateUserInfo)
			})
			r.Get("/fetch", GetUserInfo)
		})

		api.Route("/server", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Group(func(r chi.Router) {
				r.Use(httprate.LimitByIP(10, time.Minute))
				r.Post("/create", CreateServer)
				r.Post("/delete", DeleteServer)
				r.Post("/rename", RenameServer)
			})
			r.With(SessionVerifier).Get("/fetch", GetServerList)
		})

		api.Route("/channel", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Group(func(r chi.Router) {
				r.Use(httprate.LimitByIP(10, time.Second*10))
				r.Post("/create", CreateChannel)
				r.Post("/delete", nil)
				r.Post("/rename", nil)
			})
			r.With(SessionVerifier).Get("/fetch", GetChannelList)
		})

		api.Route("/message", func(r chi.Router) {
			r.Use(UserVerifier)
			r.Group(func(r chi.Router) {
				r.Use(httprate.LimitByIP(10, time.Second*10))
				r.Post("/create", CreateMessage)
				r.Post("/delete", DeleteMessage)
				r.Post("/edit", nil)
			})
			r.With(SessionVerifier).Get("/fetch", GetMessageList)
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

	//if cfg.BehindNginx {
	//	websocketPath = "/ws/"
	//} else {
	websocketPath = "/ws"
	r.Handle("/cdn/*", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))
	r.Handle("/*", http.FileServer(http.Dir("./static")))
	//}

	r.With(UserVerifier).Get(websocketPath, HandleWebSocket)

	address := fmt.Sprintf("%s:%s", cfg.HostAddress, cfg.HostPort)

	if isHttps {
		return http.ListenAndServeTLS(address, cfg.TlsCert, cfg.TlsKey, r)
	}
	return http.ListenAndServe(address, r)
}
