// Package gorp provdes a server that accepts datapoints as JSON and sends
// them to a local rserve daemon, which then processes them with Twitter's
// AnomalyDetection R package and returns the resulting PNG plot and list of
// anomalies to the client.
package gorp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/schallert/gorp/rserve"
)

// A Server defines a gorp server that accepts HTTP requests and communicates
// with the rserve client.
type Server struct {
	hsrv   *http.Server
	client rserve.Client
}

// NewServer returns a Server listening for connections on addr and
// communicating with an rserve daemon at raddr. The rserve daemon
// must be running on localhost.
func NewServer(addr, raddr string) (*Server, error) {
	client, err := rserve.NewClient(raddr)
	if err != nil {
		return nil, fmt.Errorf("r client err: %v", err)
	}

	s := &Server{
		client: client,
	}

	router := httprouter.New()
	router.HandlerFunc("POST", "/images", s.handlePostImage)
	router.HandlerFunc("GET", "/health", s.handleGetHealth)

	s.hsrv = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go s.Run()

	return s, nil
}

// Run initiates listening for HTTP requests and blocks.
func (s *Server) Run() error {
	return s.hsrv.ListenAndServe()
}

// handlePostImage handles POST requests of JSON-formatted datapoints. It
// processes them using the local rserve daemon and returns either just the PNG
// as 'image/png' or the base64-encoded PNG and an array of anomalies depending
// on the 'Accept' header.
func (s *Server) handlePostImage(w http.ResponseWriter, r *http.Request) {
	var points []rserve.Datapoint
	err := json.NewDecoder(r.Body).Decode(&points)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, fmt.Sprintf("decode post err: %v", err), http.StatusInternalServerError)
		return
	}

	res, err := s.client.GeneratePNG(points)
	if err != nil {
		http.Error(w, fmt.Sprintf("GeneratePNG err: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Gorp-Method", res.Method)
	w.Header().Set("X-Gorp-Anomalies", strconv.Itoa(len(res.Anomalies)))

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(res)
		if err != nil {
			fmt.Printf("res encode err: %v\n", err)
			return
		}
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(res.PngData)))

	if _, err := w.Write(res.PngData); err != nil {
		http.Error(w, fmt.Sprintf("write err: %v", err), http.StatusInternalServerError)
	}
}

// handleGetHealth returns the health of the server.
func (s *Server) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}
