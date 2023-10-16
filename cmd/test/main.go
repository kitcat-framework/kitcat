package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func main() {
	m := mux.NewRouter()

	go func() {
		i := 0

		for {
			m.HandleFunc(fmt.Sprintf("/%d", i), func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			i++
			time.Sleep(1 * time.Second)
		}
	}()

	if err := http.ListenAndServe(":8080", m); err != nil {
		panic(err)
	}
}
