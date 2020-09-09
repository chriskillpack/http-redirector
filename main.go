package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/kardianos/service"
)

type cfgProxy struct {
	Incoming string
	Target   string

	Cert string
	Key  string
}

type proxyEntry struct {
	URL         *url.URL
	Certificate *tls.Certificate
}

type httpsProxy struct {
	DefaultCert string     `toml:"default_cert"`
	DefaultKey  string     `toml:"default_key"`
	Entries     []cfgProxy `toml:"entry"`
}

type tomlConfig struct {
	Redirects  map[string]string
	HTTPSProxy httpsProxy `toml:"https_proxy"`
}

type program struct{}

var (
	configFile = flag.String("config", "http-redirector.toml", "Path to configuration file")
	port       = flag.Int("port", 80, "Redirect server port")
	sslPort    = flag.Int("sslport", 443, "HTTPS proxy server port")
	svcControl = flag.String("service", "", fmt.Sprintf("Service action, from %v", service.ControlAction))

	config      tomlConfig
	proxyMap    map[string]*proxyEntry
	defaultCert *tls.Certificate
	mu          sync.RWMutex

	server    *http.Server
	sslServer *http.Server
	srvWg     sync.WaitGroup
	logger    service.Logger
)

func readConfig() error {
	logger.Infof("Reading config from %s", *configFile)

	mu.Lock()
	defer mu.Unlock()

	config = tomlConfig{} // clear out config
	var err error
	if _, err = toml.DecodeFile(*configFile, &config); err != nil {
		logger.Error(err)
		return err
	}
	if config.HTTPSProxy.DefaultCert != "" && config.HTTPSProxy.DefaultKey != "" {
		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(config.HTTPSProxy.DefaultCert, config.HTTPSProxy.DefaultKey); err != nil {
			logger.Infof("Could not load cert %s", config.HTTPSProxy.DefaultCert)
			return err
		}
		defaultCert = &cert
	}

	if len(config.HTTPSProxy.Entries) > 0 {
		proxyMap = make(map[string]*proxyEntry)
		for _, ent := range config.HTTPSProxy.Entries {
			pe := proxyEntry{}

			var url *url.URL
			if url, err = url.Parse(ent.Target); err != nil {
				logger.Infof("Could not parse %s\n", ent.Target)
				return err
			}
			pe.URL = url

			if ent.Cert != "" && ent.Key != "" {
				var cert tls.Certificate
				if cert, err = tls.LoadX509KeyPair(ent.Cert, ent.Key); err != nil {
					logger.Infof("Could not load cert %s", ent.Cert)
					return err
				}
				pe.Certificate = &cert
			}

			proxyMap[ent.Incoming] = &pe
		}
	} else {
		proxyMap = nil
	}

	logger.Infof("Read %d redirects and %d proxy entries", len(config.Redirects), len(proxyMap))

	return nil
}

func startRedirector() {
	logger.Infof("Starting HTTP on port %d", *port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if redir, ok := config.Redirects[r.Host]; ok {
			w.Header().Set("Location", redir)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.NotFound(w, r)
		}
	})

	server = &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Errorf("HTTP server ListenAndServe: %v", err)
	}

	srvWg.Done()
}

func proxyGetCert(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	mu.RLock()
	defer mu.RUnlock()

	if pe, ok := proxyMap[chi.ServerName]; ok && pe.Certificate != nil {
		return pe.Certificate, nil
	}

	return defaultCert, nil
}

func startSslProxy() {
	logger.Infof("Start HTTPS proxy on port %d", *sslPort)

	rp := httputil.NewSingleHostReverseProxy(&url.URL{})
	rp.Director = func(req *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if pe, ok := proxyMap[req.Host]; ok {
			req.URL.Scheme = pe.URL.Scheme
			req.URL.Host = pe.URL.Host
		}
	}

	sslServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", *sslPort),
		Handler: rp,
		TLSConfig: &tls.Config{
			GetCertificate: proxyGetCert,
		},
	}
	if err := sslServer.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		logger.Errorf("HTTPS server ListenAndServe: %v", err)
		return
	}

	srvWg.Done()
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

	go func() {
		// Reload config when SIGHUP is received
		for {
			<-c
			if err := readConfig(); err != nil {
				logger.Error(err)
			}
		}
	}()

	srvWg.Add(2)
	go startRedirector()
	go startSslProxy()
	srvWg.Wait()
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
