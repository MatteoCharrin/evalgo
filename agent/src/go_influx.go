package main

import (
	"client/moninfluxdb"
	"log"
	"sort"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// Periodically flush in-memory metrics to InfluxDB
func goInflux(client *influxdb3.Client) {
	log.Println("[influx] goInflux started")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	wroteCPUInfo := false

	for range ticker.C {
		log.Println("[influx] tick")
		// CPU static info (once)
        if !wroteCPUInfo && Datas.CPU != nil {
            if err := moninfluxdb.WriteCPUInfo(client, *Datas.CPU); err != nil {
                log.Println("[influx] cpu_info write error:", err)
            }
            wroteCPUInfo = true
        }

		// CPU per-core usage
        if Datas.CPULoad != nil {
            if err := moninfluxdb.WriteCPUUsage(client, *Datas.CPULoad); err != nil {
                log.Println("[influx] cpu_usage write error:", err)
            }
        }

		// Memory
        if Datas.Mem != nil {
            if err := moninfluxdb.WriteMem(client, Datas.Mem); err != nil {
                log.Println("[influx] mem write error:", err)
            }
        }

		// Filesystems
        if Datas.Parts != nil {
            if err := moninfluxdb.WriteFSUsage(client, *Datas.Parts); err != nil {
                log.Println("[influx] fs_usage write error:", err)
            }
        }

		// NICs
		if Datas.Nics != nil {
			rates := *Datas.Nics
			inputs := make([]moninfluxdb.NicRateInput, 0, len(rates))
			for _, n := range rates {
				inputs = append(inputs, moninfluxdb.NicRateInput{
					Name:   n.Name,
					MTU:    n.MTU,
					Addr:   n.Addr,
					RxBps:  n.RxBps,
					TxBps:  n.TxBps,
					RxMbps: n.RxMbps,
					TxMbps: n.TxMbps,
					Up:     n.Up,
				})
			}
            if err := moninfluxdb.WriteNics(client, inputs); err != nil {
                log.Println("[influx] net_if write error:", err)
            }
        }

		// Processes (limit to top by memory percent to avoid flood)
		if Datas.Procs != nil {
			procs := append([]ProcDTO(nil), (*Datas.Procs)...)
			sort.SliceStable(procs, func(i, j int) bool { return procs[i].MemoryPercent > procs[j].MemoryPercent })
			const maxN = 20
			if len(procs) > maxN {
				procs = procs[:maxN]
			}
			inputs := make([]moninfluxdb.ProcInput, 0, len(procs))
			for _, p := range procs {
				inputs = append(inputs, moninfluxdb.ProcInput{
					PID:           p.PID,
					Name:          p.Name,
					Status:        p.Status,
					Username:      p.Username,
					NumThreads:    p.NumThreads,
					MemoryRSS:     p.MemoryRSS,
					MemoryVMS:     p.MemoryVMS,
					MemoryPercent: p.MemoryPercent,
					CreateTime:    p.CreateTime,
				})
			}
            if err := moninfluxdb.WriteProcs(client, inputs); err != nil {
                log.Println("[influx] proc write error:", err)
            }
        }

		LogMessage("goroutine: goInflux")
	}
}
