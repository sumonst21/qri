package server

import (
	"fmt"
	"github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/qri/core/datasets"
	"github.com/qri-io/qri/core/graphs"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type Server struct {
	cfg   *Config
	log   *logrus.Logger
	ns    map[string]datastore.Key
	store *ipfs.Datastore
}

func New(options ...func(*Config)) (*Server, error) {
	cfg := DefaultConfig()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		// panic if the server is missing a vital configuration detail
		return nil, fmt.Errorf("server configuration error: %s", err.Error())
	}

	ns := graphs.LoadNamespaceGraph(cfg.NamespaceGraphPath)

	s := &Server{
		cfg: cfg,
		log: logrus.New(),

		ns: ns,
	}

	// output to stdout in dev mode
	if s.cfg.Mode == DEVELOP_MODE {
		s.log.Out = os.Stdout
	} else {
		s.log.Out = os.Stderr
	}
	s.log.Level = logrus.InfoLevel
	s.log.Formatter = &logrus.TextFormatter{
		ForceColors: true,
	}

	return s, nil
}

// main app entry point
func (s *Server) Serve() error {
	store, err := ipfs.NewDatastore(func(cfg *ipfs.StoreCfg) {
		cfg.Online = true
	})
	if err != nil {
		return err
	}
	s.store = store

	server := &http.Server{}
	server.Handler = s.NewServerRoutes()

	// fire it up!
	s.log.Println("starting server on port", s.cfg.Port)

	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, server)
}

// NewServerRoutes returns a Muxer that has all API routes.
// This makes for easy testing using httptest, see server_test.go
func (s *Server) NewServerRoutes() *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/", apiutil.NotFoundHandler)
	m.Handle("/status", s.middleware(apiutil.HealthCheckHandler))

	dsh := datasets.NewHandlers(s.store, s.ns)
	m.Handle("/datasets", s.middleware(dsh.ListHandler))
	m.Handle("/datasets/", s.middleware(dsh.GetHandler))

	return m
}
