// moninfluxdb.go
package moninfluxdb

import (
	"context"
	"strconv"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// --- identité de l'hôte (tag Influx) ---
// Par défaut "localhost", mais l'agent doit appeler SetHostID(...) au démarrage.
var hostTag = "localhost"

// SetHostID permet de définir l'identité stable (ex: HOST_ID ou hostname)
func SetHostID(id string) {
	if id != "" {
		hostTag = id
	}
}

// Open ouvre un client InfluxDB v3
func Open(host, name, token string) (*influxdb3.Client, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     host,
		Token:    token,
		Database: name,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

// WriteLoad écrit les load averages
func WriteLoad(client *influxdb3.Client, mstats *load.AvgStat) error {
	if client == nil || mstats == nil {
		return nil
	}
	now := time.Now().UTC()
	pt := influxdb3.NewPoint(
		"load_avg",
		map[string]string{"host_id": hostTag},
		map[string]any{
			"load1":  mstats.Load1,
			"load5":  mstats.Load5,
			"load15": mstats.Load15,
		},
		now,
	)
	return client.WritePoints(context.Background(), []*influxdb3.Point{pt})
}

// WriteCPUUsage écrit l’utilisation CPU par cœur (un point par core)
func WriteCPUUsage(client *influxdb3.Client, percs []float64) error {
	if client == nil || len(percs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	pts := make([]*influxdb3.Point, 0, len(percs))
	for i, v := range percs {
		pts = append(pts, influxdb3.NewPoint(
			"cpu_usage",
			map[string]string{
				"host_id": hostTag,
				"core":    itoa(i),
			},
			map[string]any{"percent": v},
			now,
		))
	}
	return client.WritePoints(context.Background(), pts)
}

// WriteCPUInfo écrit les infos CPU statiques (model, cores, mhz) par entrée
func WriteCPUInfo(client *influxdb3.Client, info []cpu.InfoStat) error {
	if client == nil || len(info) == 0 {
		return nil
	}
	now := time.Now().UTC()
	pts := make([]*influxdb3.Point, 0, len(info))
	for i, c := range info {
		pts = append(pts, influxdb3.NewPoint(
			"cpu_info",
			map[string]string{
				"host_id": hostTag,
				"index":   itoa(i),
			},
			map[string]any{
				"model": c.ModelName,
				"cores": c.Cores,
				"mhz":   c.Mhz,
			},
			now,
		))
	}
	return client.WritePoints(context.Background(), pts)
}

// WriteMem écrit les statistiques mémoire
func WriteMem(client *influxdb3.Client, v *mem.VirtualMemoryStat) error {
	if client == nil || v == nil {
		return nil
	}
	now := time.Now().UTC()
	pt := influxdb3.NewPoint(
		"mem",
		map[string]string{"host_id": hostTag},
		map[string]any{
			"total":        v.Total,
			"used":         v.Used,
			"free":         v.Free,
			"available":    v.Available,
			"used_percent": v.UsedPercent,
			// champs additionnels possibles si tu veux: cached, buffers, active, inactive, slab...
		},
		now,
	)
	return client.WritePoints(context.Background(), []*influxdb3.Point{pt})
}

// WriteFSUsage écrit l’usage par système de fichiers monté
func WriteFSUsage(client *influxdb3.Client, parts []disk.PartitionStat) error {
	if client == nil || len(parts) == 0 {
		return nil
	}
	now := time.Now().UTC()
	pts := make([]*influxdb3.Point, 0, len(parts))
	for _, p := range parts {
		// filtrage systèmes pseudo-fs
		switch p.Fstype {
		case "proc", "sysfs", "devtmpfs", "devpts", "overlay":
			continue
		}
		if du, err := disk.Usage(p.Mountpoint); err == nil && du != nil {
			pts = append(pts, influxdb3.NewPoint(
				"fs_usage",
				map[string]string{
					"host_id": hostTag,
					"mount":   p.Mountpoint,
					"fstype":  p.Fstype,
				},
				map[string]any{
					"total":        du.Total,
					"used":         du.Used,
					"free":         du.Free,
					"used_percent": du.UsedPercent,
				},
				now,
			))
		}
	}
	if len(pts) == 0 {
		return nil
	}
	return client.WritePoints(context.Background(), pts)
}

// --- structures d’entrée complémentaires ---
type NicRateInput struct {
	Name   string
	MTU    int
	Addr   []string
	RxBps  float64
	TxBps  float64
	RxMbps float64
	TxMbps float64
	Up     bool
}

type ProcInput struct {
	PID           int32
	Name          string
	Status        string
	Username      string
	NumThreads    int32
	MemoryRSS     uint64
	MemoryVMS     uint64
	MemoryPercent float32
	CreateTime    int64
}

// WriteNics écrit les débits par interface réseau
func WriteNics(client *influxdb3.Client, nics []NicRateInput) error {
	if client == nil || len(nics) == 0 {
		return nil
	}
	now := time.Now().UTC()
	pts := make([]*influxdb3.Point, 0, len(nics))
	for _, n := range nics {
		pts = append(pts, influxdb3.NewPoint(
			"net_if",
			map[string]string{
				"host_id": hostTag,
				"name":    n.Name,
			},
			map[string]any{
				"mtu":     n.MTU,
				"up":      n.Up,
				"rx_bps":  n.RxBps,
				"tx_bps":  n.TxBps,
				"rx_mbps": n.RxMbps,
				"tx_mbps": n.TxMbps,
			},
			now,
		))
	}
	return client.WritePoints(context.Background(), pts)
}

// WriteProcs écrit des stats légères par process (utilisation raisonnée)
func WriteProcs(client *influxdb3.Client, procs []ProcInput) error {
	if client == nil || len(procs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	pts := make([]*influxdb3.Point, 0, len(procs))
	for _, p := range procs {
		pts = append(pts, influxdb3.NewPoint(
			"proc",
			map[string]string{
				"host_id": hostTag,
				"pid":     strconv.FormatInt(int64(p.PID), 10),
			},
			map[string]any{
				"name":           p.Name,
				"status":         p.Status,
				"username":       p.Username,
				"threads":        p.NumThreads,
				"memory_rss":     p.MemoryRSS,
				"memory_vms":     p.MemoryVMS,
				"memory_percent": p.MemoryPercent,
				"create_time":    p.CreateTime,
			},
			now,
		))
	}
	return client.WritePoints(context.Background(), pts)
}

// helpers
func itoa(i int) string { return strconv.FormatInt(int64(i), 10) }
