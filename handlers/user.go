package handlers

import (
	"chatapp-backend/utils/jwt"
	"fmt"
	"net/http"
	"strconv"
)

func User(userToken jwt.UserToken, w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")

	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	fmt.Fprintln(w, id)
}
