package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

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

	config tomlConfig
	server *http.Server
)

func (p *program) Start(s service.Service) error {
	log.Printf("Reading config from %s", *configFile)
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		log.Fatal(err)
	}
	log.Printf("Read %d redirects", len(config.Redirects))

	go p.run()
	return nil
}

func (p *program) run() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if redir, ok := config.Redirects[r.Host]; ok {
			w.Header().Set("Location", redir)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.NotFound(w, r)
		}
	})
	log.Printf("Starting server on port %d", *port)

	server = &http.Server{Addr: fmt.Sprintf(":%d", *port)}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}

func (p *program) Stop(s service.Service) error {
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

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
		if err := service.Control(s, *svcControl); err != nil {
			log.Fatal(err)
		}

		return
	}

	s.Run()
}
