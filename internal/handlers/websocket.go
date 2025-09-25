package handlers

import (
	"chatapp-backend/internal/hub"
	"net/http"
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(uint64)

	hub.HandleClient(w, r, userID)
}
