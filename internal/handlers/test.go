package handlers

import (
	"fmt"
	"net/http"
)

func Test(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprint(w, "Hello world!")
	if err != nil {
		sugar.Error(err)
		return
	}
}
