// Command volra-proxy is the A2A smart sidecar reverse proxy.
// It runs as a Docker container alongside the agent, handling A2A task execution.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/proxy"
)

func main() {
	agentURL := envOrDefault("VOLRA_AGENT_URL", "http://agent:8000")
	cardPath := envOrDefault("VOLRA_CARD_PATH", "/etc/volra/agent-card.json")
	modeStr := envOrDefault("VOLRA_A2A_MODE", "default")
	listenAddr := envOrDefault("VOLRA_LISTEN_ADDR", ":80")

	mode := agentfile.A2AMode(modeStr)

	var skills []agentfile.A2ASkill
	if raw := os.Getenv("VOLRA_A2A_SKILLS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &skills); err != nil {
			log.Fatalf("parsing VOLRA_A2A_SKILLS: %v", err)
		}
	}

	caller := proxy.NewAgentCaller(agentURL, mode, skills)
	handler, err := proxy.NewHandler(agentURL, cardPath, caller)
	if err != nil {
		log.Fatalf("creating proxy handler: %v", err)
	}

	fmt.Printf("volra-proxy starting on %s (mode=%s, agent=%s)\n", listenAddr, mode, agentURL)
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
