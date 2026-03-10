package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/romerox3/volra/internal/console"
)

// Server is the control plane HTTP server.
type Server struct {
	store      *Store
	federation *FederationClient
	mux        *http.ServeMux
	httpServer *http.Server
	port       int
}

// NewServer creates a new control plane server.
func NewServer(store *Store, port int) *Server {
	s := &Server{
		store:      store,
		federation: NewFederationClient(),
		mux:        http.NewServeMux(),
		port:       port,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/agents/{name}", s.handleGetAgent)
	s.mux.HandleFunc("POST /api/agents/{name}/deploy", s.handleDeployAgent)
	s.mux.HandleFunc("POST /api/agents/{name}/stop", s.handleStopAgent)
	s.mux.HandleFunc("GET /api/federation/capabilities", s.handleFederationCapabilities)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Serve console UI at root.
	s.mux.Handle("/", console.Handler())
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listening on port %d: %w", s.port, err)
	}

	log.Printf("Control plane listening on :%d", s.port)
	return s.httpServer.Serve(ln)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the HTTP handler (for testing).
func (s *Server) Handler() http.Handler {
	return s.mux
}

// --- Handlers ---

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing agents: "+err.Error())
		return
	}
	if agents == nil {
		agents = []Agent{}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	agent, err := s.store.GetAgent(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying agent: "+err.Error())
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleDeployAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	agent, err := s.store.GetAgent(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying agent: "+err.Error())
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}

	// Mark as deploying — actual deploy will be async in the future.
	_ = s.store.UpdateAgentStatus(name, "deploying")

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": fmt.Sprintf("deploy triggered for agent %s", name),
	})
}

func (s *Server) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	agent, err := s.store.GetAgent(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "querying agent: "+err.Error())
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}

	_ = s.store.UpdateAgentStatus(name, "stopped")

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("agent %s stopped", name),
	})
}

func (s *Server) handleFederationCapabilities(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing agents: "+err.Error())
		return
	}
	if agents == nil {
		agents = []Agent{}
	}

	peers, err := s.store.ListPeers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing peers: "+err.Error())
		return
	}

	caps := s.federation.FetchCapabilities(r.Context(), agents, peers, "")
	if caps == nil {
		caps = []FederatedCapability{}
	}
	writeJSON(w, http.StatusOK, caps)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
