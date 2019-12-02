package main

import (
	"log"
	"net/http"
)

type redirect struct {
	reqHost string
	redir   string
}

var redirects = []redirect{
	{"eg", "https://google.com"},
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		for _, red := range redirects {
			if red.reqHost == host {
				w.Header().Set("Location", red.redir)
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
		}
		http.NotFound(w, r)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
