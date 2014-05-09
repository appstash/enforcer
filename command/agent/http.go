package agent

import (
//	"fmt"
	"bytes"
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"net"
	"net/http"
	"net/http/pprof"
//	"os"
	"time"
	"io"
	"strconv"
	"log"
)

// HTTPServer is used to wrap an Agent and expose various API's
// in a RESTful manner
type HTTPServer struct {
	agent *Agent
	mux *http.ServeMux
	listener net.Listener
	logger *log.Logger
	uiDir string
}

// NewHTTPServer starts a new HTTP server to provide an interface to
// the agent.
func NewHTTPServer(agent *Agent, uiDir string, enableDebug bool, logOutput io.Writer, bind string) (*HTTPServer, error) {
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
		uiDir:	  uiDir,
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


func (s *HTTPServer) registerHandlers(enableDebug bool) {
	s.mux.HandleFunc("/", s.Index)

	//s.mux.HandleFunc("/v1/agent/members", s.wrap(s.AgentMembers))

	//s.mux.HandleFunc("/v1/ipxe/nodes", s.wrap(s.Nodes))
	//s.mux.HandleFunc("/v1/ipxe/{id}", s.wrap(s.NodeIpxe))
	//s.mux.HandleFunc("/v1/ipxe/toggle", s.wrap(s.ToggleNode))


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
		/*s.mux.HandleFunc("/v1/internal/ui/nodes", s.wrap(s.UINodes))
		s.mux.HandleFunc("/v1/internal/ui/node/", s.wrap(s.UINodeInfo))
		s.mux.HandleFunc("/v1/internal/ui/services", s.wrap(s.UIServices))*/
	}
}


// wrap is used to wrap functions to make them more convenient
func (s *HTTPServer) wrap(handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	f := func(resp http.ResponseWriter, req *http.Request) {
		// Invoke the handler
		start := time.Now()
		defer func() {
			s.logger.Printf("[DEBUG] http: Request %v (%v)", req.URL, time.Now().Sub(start))
		}()
		obj, err := handler(resp, req)

		// Check for an error
	HAS_ERR:
		if err != nil {
			s.logger.Printf("[ERR] http: Request %v, error: %v", req.URL, err)
			resp.WriteHeader(500)
			resp.Write([]byte(err.Error()))
			return
		}

		// Write out the JSON object
		if obj != nil {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			if err = enc.Encode(obj); err != nil {
				goto HAS_ERR
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf.Bytes())
		}
	}
	return f
}

// Renders a simple index page
func (s *HTTPServer) Index(resp http.ResponseWriter, req *http.Request) {
	// Check if this is a non-index path
	if req.URL.Path != "/" {
		resp.WriteHeader(404)
		return
	}

	// Check if we have no UI configured
	if s.uiDir == "" {
		resp.Write([]byte("Consul Agent"))
		return
	}

	// Redirect to the UI endpoint
	http.Redirect(resp, req, "/ui/", 301)
}

// decodeBody is used to decode a JSON request body
func decodeBody(req *http.Request, out interface{}, cb func(interface{}) error) error {
	var raw interface{}
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	// Invoke the callback prior to decode
	if cb != nil {
		if err := cb(raw); err != nil {
			return err
		}
	}
	return mapstructure.Decode(raw, out)
}

// setIndex is used to set the index response header
func setIndex(resp http.ResponseWriter, index uint64) {
	resp.Header().Add("X-Consul-Index", strconv.FormatUint(index, 10))
}
