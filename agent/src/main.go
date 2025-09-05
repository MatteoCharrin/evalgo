package main

import (
	"client/moninfluxdb"
	"fmt"
	"net/http"
	"os"
)

var HostID = getenv("HOST_ID", "")

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

const DBHost string = "http://172.16.4.78:8181"
const DBName string = "metrics"
const DBToken string = "apiv3_9rdZKWDNT9DcuIli9W6R9ePr9KpeRXTkWMktG_1B5r504DRNMsYviUUz5107Hq1K9C4xmIvJDo7YayM1nd_Wsg"
const ServerURL string = "http://172.16.4.78:8080"
const DEBUG bool = true

func main() {
	// Connexion InfluxDB
	client, err := moninfluxdb.Open(DBHost, DBName, DBToken)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	// main.go (agent) — juste après avoir obtenu le client Influx
	host, _ := os.Hostname()
	if HostID == "" {
		HostID = host
	}
	// pousser l'ID côté lib influx
	moninfluxdb.SetHostID(HostID)
	// MaJ de AllDatas
	go goLoad(client)
	go goCPU()
	go goDisk()
	go goProcs()
	go goNics()
	go goMem()
	// Flush vers InfluxDB pour toutes les métriques
	go goInflux(client)

	http.HandleFunc("GET /cpu", webcpu)
	http.HandleFunc("GET /cpu/{id}", webcpubyid)
	http.HandleFunc("GET /cpu/load", webcpuload)
	http.HandleFunc("GET /load", webload)
	http.HandleFunc("GET /procs", webprocs)
	http.HandleFunc("GET /procs/{id}", webprocsbypid)
	// Kill process by pid (simple GET)
	http.HandleFunc("GET /procs/kill/{pid}", webprocskill)
	http.HandleFunc("GET /disks", webdisk)
	http.HandleFunc("GET /nics", webnics)
	http.HandleFunc("GET /mem", webmem)
	fmt.Println("Serveur :8080")
	http.ListenAndServe(":8080", nil)
}
