// web_procs.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// NOTE : on suppose que ces symboles existent ailleurs dans ton projet.
// - Datas.Procs (liste des process côté DTO)
// - DTOProcLoad(id int32) (retourne le détail d’un process)
//// type Example only if you need a stub:
// var Datas struct{ Procs any }
// func DTOProcLoad(id int32) (any, error) { return nil, nil }

// ---- Helpers CORS & JSON ----

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

// ---- Handlers ----

// GET /procs
func webprocs(w http.ResponseWriter, req *http.Request) {
	setCORS(w)
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Datas.Procs doit être sérialisable (slice/struct, etc.)
	writeJSON(w, http.StatusOK, Datas.Procs)
	log.Println("GET /procs")
}

// GET /procs/{id}
func webprocsbypid(w http.ResponseWriter, req *http.Request) {
	setCORS(w)
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	idStr := req.PathValue("id")
	idint, err := strconv.Atoi(idStr)
	if err != nil || idint < 0 {
		http.Error(w, "Erreur: id invalide", http.StatusBadRequest)
		return
	}
	id := int32(idint)

	out, err := DTOProcLoad(id)
	if err != nil {
		http.Error(w, "Erreur dans le processus", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, out)
	log.Printf("GET /procs/%d\n", id)
}

// GET /procs/kill/{pid}
func webprocskill(w http.ResponseWriter, req *http.Request) {
	setCORS(w)
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	pidStr := req.PathValue("pid")
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		http.Error(w, "pid invalide", http.StatusBadRequest)
		return
	}

	// Version portable: fonctionne aussi sous Windows (pas de syscall.Kill)
	proc, err := os.FindProcess(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("process introuvable: %v", err), http.StatusNotFound)
		return
	}
	if err := proc.Kill(); err != nil {
		http.Error(w, fmt.Sprintf("échec kill: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "killed %d", pid)
	log.Println("GET /procs/kill", pid)
}
