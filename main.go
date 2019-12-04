package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	Redirects map[string]string
}

func main() {
	configFile := flag.String("config", "config.toml", "Path to configuration file")
	port := flag.Int("port", 80, "Redirect server port")
	flag.Parse()

	log.Printf("Reading config from %s", *configFile)
	var config tomlConfig
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("Read %d redirects", len(config.Redirects))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if redir, ok := config.Redirects[r.Host]; ok {
			w.Header().Set("Location", redir)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.NotFound(w, r)
		}
	})
	log.Printf("Starting server on port %d", *port)

	var server *http.Server

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Println("SIGINT received, shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	server = &http.Server{Addr: fmt.Sprintf(":%d", *port)}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
