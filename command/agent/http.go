package agent

import (
	"fmt"
	"net"
	"os"
	"time"
)

// HTTPServer is used to wrap an Agent and expose various API's
// in a RESTful manner
type HTTPServer struct {
	agent *Agent
	mux *http.ServeMux
	listener net.Listener
	logger *log.Logger
}

// NewHTTPServer starts a new HTTP server to provide an interface to
// the agent.
func NewHTTPServer(agent *Agent, enableDebug bool, logOutput io.Writer, bind string) (*HTTPServer, error) {
	// Create the mux.
	mux := http.NewServeMux()

	// Create listener
	list, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}


	// Create the server
	srv := &HTTPServer{
		agent:    agent,
		mux:      mux,
		listener: list,
		logger:   log.New(logOutput, "", log.LstdFlags),
	}
	srv.registerHandlers(enableDebug)

	// Start the server
	go http.Serve(list, mux)
	return srv, nil
}

// Shutdown is used to shutdown the HTTP server
func (s *HTTPServer) Shutdown() {
	s.listener.Close()
}


func (s *HTTPServer) Handlers (enableDebug, bool) {
	s.mux.HandleFunc("/", s.Index)

	s.mux.HandleFunc("/v1/agent/members", s.wrap(s.AgentMembers))

	s.mux.HandleFunc("/v1/ipxe/nodes"), s.wrap(s.IpxeNodes))
	s.mux.HandleFunc("/v1/ipxe/{id}", s.wrap(s.IpxeId))


	if enableDebug {
		s.mux.HandleFunc("/debug/pprof/", pprof.Index)
		s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	// Enable the UI + special endpoints
	if s.uiDir != "" {
		// Static file serving done from /ui/
		s.mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(s.uiDir))))

		// API's are under /internal/ui/ to avoid conflict
		s.mux.HandleFunc("/v1/internal/ui/nodes", s.wrap(s.UINodes))
		s.mux.HandleFunc("/v1/internal/ui/node/", s.wrap(s.UINodeInfo))
		s.mux.HandleFunc("/v1/internal/ui/services", s.wrap(s.UIServices))
	}
}
