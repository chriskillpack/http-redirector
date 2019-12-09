package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kardianos/service"
)

type tomlConfig struct {
	Redirects map[string]string
}

type program struct{}

var (
	configFile = flag.String("config", "config.toml", "Path to configuration file")
	port       = flag.Int("port", 80, "Redirect server port")
	svcControl = flag.String("service", "", fmt.Sprintf("Service action, from %v", service.ControlAction))
)

func (p *program) Start(s service.Service) error {
	log.Printf("***** in the Start")
	go p.run()
	return nil
}

func (p *program) run() {
	log.Printf("***** in the run")
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "http-redirector",
		DisplayName: "HTTP-redirector",
		Description: "Redirects HTTP traffic on the local LAN",
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	flag.Parse()

	if *svcControl != "" {
		svc := strings.ToLower(*svcControl)

		ok := false
		for _, ca := range service.ControlAction {
			if ca == svc {
				ok = true
				break
			}
		}
		if !ok {
			log.Fatalf("Unknown service action %s", *svcControl)
		}

		if err := service.Control(s, svc); err != nil {
			log.Fatal(err)
		}

		return
	}

	log.Printf("Reading config from %s", *configFile)
	var config tomlConfig
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		log.Fatal(err)
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
