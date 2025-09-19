package handlers

import "net/http"

func Test(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello World"))
	if err != nil {
		sugar.Error(err)
		return
	}
}
