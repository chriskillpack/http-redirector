package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/kardianos/service"
)

type tomlConfig struct {
	Redirects map[string]string
}

type program struct{}

var (
	configFile = flag.String("config", "http-redirector.toml", "Path to configuration file")
	port       = flag.Int("port", 80, "Redirect server port")
	svcControl = flag.String("service", "", fmt.Sprintf("Service action, from %v", service.ControlAction))

	config tomlConfig
	mu     sync.RWMutex

	server *http.Server
	logger service.Logger
)

func readConfig() error {
	logger.Infof("Reading config from %s", *configFile)

	mu.Lock()
	defer mu.Unlock()

	config = tomlConfig{} // clear out config
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		logger.Error(err)
		return err
	}
	logger.Infof("Read %d redirects", len(config.Redirects))

	return nil
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running from terminal")
	} else {
		logger.Info("Running under service manager")
	}

	if err := readConfig(); err != nil {
		return err
	}

	go p.run()
	return nil
}

func (p *program) run() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if redir, ok := config.Redirects[r.Host]; ok {
			w.Header().Set("Location", redir)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.NotFound(w, r)
		}
	})
	logger.Infof("Starting server on port %d", *port)

	go func() {
		// Reload config when SIGHUP is received
		for {
			<-c
			if err := readConfig(); err != nil {
				logger.Error(err)
			}
		}
	}()

	server = &http.Server{Addr: fmt.Sprintf(":%d", *port)}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Errorf("HTTP server ListenAndServe: %v", err)
		return
	}
}

func (p *program) Stop(s service.Service) error {
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Infof("HTTP server Shutdown: %v", err)
	}

	return nil
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to get path to current directory")
	}
	flag.Parse()

	svcConfig := &service.Config{
		Name:             "http-redirector",
		DisplayName:      "HTTP-redirector",
		Description:      "Redirects HTTP traffic on the local LAN",
		WorkingDirectory: pwd,
		Arguments:        []string{"-config", *configFile},
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if logger, err = s.Logger(nil); err != nil {
		log.Fatal(err)
	}

	if *svcControl != "" {
		if err := service.Control(s, *svcControl); err != nil {
			log.Fatal(err)
		}

		return
	}

	s.Run()
}
