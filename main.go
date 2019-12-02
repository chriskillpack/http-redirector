package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	Redirects map[string]string
}

func main() {
	configFile := flag.String("config", "config.toml", "Path to configuration file")
	flag.Parse()

	var config tomlConfig
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if redir, ok := config.Redirects[r.Host]; ok {
			w.Header().Set("Location", redir)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.NotFound(w, r)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
