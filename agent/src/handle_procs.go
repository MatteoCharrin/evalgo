package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"syscall"
)

func webprocs(w http.ResponseWriter, req *http.Request) {

	j, _ := json.Marshal(Datas.Procs)
	// Active CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
	log.Println("/procs")
}

func webprocsbypid(w http.ResponseWriter, req *http.Request) {
	idStr := req.PathValue("id")
	idint, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Fprintf(w, "Erreur dans le processus")
		return
	}
	id := int32(idint)
	out, err := DTOProcLoad(id)
	if err != nil {
		fmt.Fprintf(w, "Erreur dans le processus")
		return
	}
	j, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
	log.Println("/procs/ID")
}

// webprocskill: kill process {pid} simply (GET)
func webprocskill(w http.ResponseWriter, req *http.Request) {
	pidStr := req.PathValue("pid")
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		fmt.Fprintf(w, "pid invalide")
		return
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		fmt.Fprintf(w, "erreur: %v", err)
		return
	}
	fmt.Fprintf(w, "killed %d", pid)
	log.Println("/procs/kill", pid)
	log.Println("Test")
}
