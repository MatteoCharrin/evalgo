package main

import (
	"log"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

func goMem() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		v, err := mem.VirtualMemory()

		if err != nil {
			log.Println("Erreur dans la memoire")
		}

		Datas.Mem = v
		LogMessage("goroutine: goMem")
	}
}
