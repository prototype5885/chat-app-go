package handlers

import (
	"chatapp-backend/utils/snowflake"
	"fmt"
	"net/http"

	"github.com/vmihailenco/msgpack/v5"
)

func Test(w http.ResponseWriter, r *http.Request) {
	type Test struct {
		First  []uint64 `msgpack:"first"`
		Second []string `msgpack:"second"`
	}

	test := Test{}

	for range 10000 {
		id, _ := snowflake.Generate()
		test.First = append(test.First, id)
		test.Second = append(test.Second, fmt.Sprint(id))
	}

	err := msgpack.NewEncoder(w).Encode(test)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
