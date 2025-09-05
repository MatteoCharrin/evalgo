package moninfluxdb

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/shirou/gopsutil/v4/load"
)

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

func WriteLoad(client *influxdb3.Client, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var mstats load.AvgStat
	if err := json.NewDecoder(resp.Body).Decode(&mstats); err != nil {
		return err
	}

	pt := influxdb3.NewPoint(
		"load_avg",                             // measurement
		map[string]string{"host": "localhost"}, // tags
		map[string]any{ // fields
			"load1":  mstats.Load1,
			"load5":  mstats.Load5,
			"load15": mstats.Load15,
		},
		time.Now().UTC(), // timestamp d’échantillonnage
	)
	if err := client.WritePoints(context.Background(), []*influxdb3.Point{pt}); err != nil {
		return err
	}
	return nil
}
