package handlers

import (
	"fmt"
	"net/http"
)

func Test(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprint(w, "Hello world!")
	if err != nil {
		sugar.Error(err)
		return
	}
}

func TestSqlite(w http.ResponseWriter, r *http.Request) {
	result, err := db.Exec("INSERT INTO testusers (name, email) VALUES (?, ?)", r.URL.Query().Get("name"), r.URL.Query().Get("email"))
	if err != nil {
		sugar.Error(err)
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		sugar.Error(err)
		return
	}

	_, err = fmt.Fprintf(w, "Inserted row with ID %d", lastID)
	if err != nil {
		sugar.Error(err)
		return
	}
}
