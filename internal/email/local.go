package email

import (
	"chatapp-backend/internal/keyValue"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type ConfirmLink struct {
	Email string
	Link  string
}

const emailConfirmations string = "email_confirmations"

func localhostListener() {
	r := chi.NewRouter()

	r.HandleFunc("/emails_to_confirm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		var htmlString []byte

		result, err := keyValue.Get(emailConfirmations)
		if err != nil {
			return
		}

		var confirmLinks []ConfirmLink
		if result != "" {
			err = json.Unmarshal([]byte(result), &confirmLinks)
			if err != nil {
				return
			}
			htmlString = fmt.Append(htmlString, "<h1>Emails waiting to be confirmed:</h1>")
			for _, link := range confirmLinks {
				htmlString = fmt.Appendf(htmlString, `<a href="%s">%s</a></br></p>`, link.Link, link.Email)
			}
		} else {
			htmlString = fmt.Appendf(htmlString, "<h1>No emails to confirm</h1>\n")
		}

		_, err = w.Write(htmlString)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	})

	localAddress := "127.0.0.1:3010"
	fmt.Printf("View email confirmation links on http://%s/emails_to_confirm\n", localAddress)
	err := http.ListenAndServe(localAddress, r)
	if err != nil {
		fmt.Println(err)
	}
}

func storeManual(email string, link string) error {
	result, err := keyValue.Get(emailConfirmations)
	if err != nil {
		return err
	}

	var confirmLinks []ConfirmLink
	if len(result) != 0 {
		err = json.Unmarshal([]byte(result), &confirmLinks)
		if err != nil {
			return err
		}
	}

	confirmLinks = append(confirmLinks, ConfirmLink{email, link})

	jsonBytes, err := json.Marshal(confirmLinks)
	if err != nil {
		return err
	}

	err = keyValue.Set(emailConfirmations, string(jsonBytes), time.Hour*1)
	if err != nil {
		return err
	}
	return nil
}
