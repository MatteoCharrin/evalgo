// multi.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var agentsStatic = []string{
  "http://172.16.4.78:8080",
  "http://172.16.1.46:8080",
  "http://172.16.4.80:8080",
}

func init() {
	// Endpoints d'agrégation (unifie la vue multi-agents)
	http.HandleFunc("GET /api/cpus",   aggregate("/cpu"))
	http.HandleFunc("GET /api/memory", aggregate("/mem"))
	http.HandleFunc("GET /api/disks",  aggregate("/disks"))
	http.HandleFunc("GET /api/nics",   aggregate("/nics"))
	http.HandleFunc("GET /api/load",   aggregate("/load"))
	http.HandleFunc("GET /api/procs",  aggregate("/procs"))

	// Proxy kill :9090 -> agent ciblé
	http.HandleFunc("GET /api/agents/{id}/procs/kill/{pid}", proxyKill)

	// Charge AGENTS via variable d'environnement (sinon tu peux coder en dur)
	if v := strings.TrimSpace(os.Getenv("AGENTS")); v != "" {
		agentsStatic = splitCSV(v)
	}
}

func aggregate(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agents := currentAgents(r)

		type item struct {
			ID   string `json:"id"`
			Host string `json:"host"`
			URL  string `json:"url"`
			Data any    `json:"data,omitempty"`
			Err  string `json:"err,omitempty"`
		}
		out := make([]item, 0, len(agents))

		// Timeout court pour ne pas bloquer si un agent est down
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		for _, base := range agents {
			url := strings.TrimRight(base, "/") + path
			it := item{ID: agentID(base), Host: hostPart(base), URL: base}
			var body any
			if err := getJSON(ctx, url, &body); err != nil {
				it.Err = err.Error()
			} else {
				it.Data = body
			}
			out = append(out, it)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func proxyKill(w http.ResponseWriter, r *http.Request) {
	id  := r.PathValue("id")
	pid := r.PathValue("pid")

	base, ok := findAgentBaseByID(id, currentAgents(r))
	if !ok {
		http.Error(w, "unknown agent id", http.StatusNotFound)
		return
	}
	target := strings.TrimRight(base, "/") + "/procs/kill/" + pid // route déjà côté agent:contentReference[oaicite:3]{index=3}

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "agent unreachable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func currentAgents(r *http.Request) []string {
	agents := agentsStatic
	if qs := strings.TrimSpace(r.URL.Query().Get("agents")); qs != "" {
		agents = splitCSV(qs) // override ponctuel: /api/cpus?agents=http://a:8080,http://b:8080
	}
	return agents
}

func splitCSV(s string) []string {
	ps := strings.Split(s, ",")
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		u := strings.TrimSpace(p)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func agentID(base string) string { // id = "host:port"
	if h := hostPart(base); h != "" { return h }
	return base
}

func hostPart(base string) string {
	s := strings.TrimPrefix(base, "http://")
	s = strings.TrimPrefix(s, "https://")
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	return s
}

func getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil { return err }
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 1 * time.Second}).DialContext,
		},
	}
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func findAgentBaseByID(id string, agents []string) (string, bool) {
	for _, a := range agents {
		if agentID(a) == id { return a, true }
	}
	return "", false
}
