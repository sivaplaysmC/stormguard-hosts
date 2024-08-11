package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func main() {
	r := chi.NewRouter()

	// Shared metrics and mutex
	var metrics SystemMetrics
	var metricsMutex sync.Mutex

	// Periodically update the metrics every 10 seconds
	go func() {
		for {
			newMetrics, err := getMetrics()
			if err != nil {
				log.Printf("Error getting metrics: %v", err)
			} else {
				metricsMutex.Lock()
				metrics = newMetrics
				metricsMutex.Unlock()
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Define the endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		metricsMutex.Lock()
		defer metricsMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Start the server
	log.Println("Starting server on :7080")
	if err := http.ListenAndServe(":7080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

type SystemMetrics struct {
	Time          string  `json:"time"`
	CPUPercent    float64 `json:"cpu_perc"`
	MemoryPercent float64 `json:"memory_perc"`
	RxRate        uint64  `json:"rx_rate"`
	TxRate        uint64  `json:"tx_rate"`
	RxBytes       uint64  `json:"rx_bytes"`
	TxBytes       uint64  `json:"tx_bytes"`
}

func getMetrics() (SystemMetrics, error) {
	// Get current time
	currentTime := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

	// Get CPU percentage
	cpuPercents, err := cpu.Percent(0, false)
	if err != nil {
		return SystemMetrics{}, err
	}
	cpuPercent := cpuPercents[0]

	// Get memory usage
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return SystemMetrics{}, err
	}
	memoryPercent := vmem.UsedPercent

	// Get network IO counters
	netIO, err := net.IOCounters(false)
	if err != nil {
		return SystemMetrics{}, err
	}
	rxBytes := netIO[0].BytesRecv
	txBytes := netIO[0].BytesSent

	// Calculate network rates
	rxRate := netIO[0].PacketsRecv // packets received per interval
	txRate := netIO[0].PacketsSent // packets sent per interval

	metrics := SystemMetrics{
		Time:          currentTime,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		RxRate:        rxRate,
		TxRate:        txRate,
		RxBytes:       rxBytes,
		TxBytes:       txBytes,
	}

	return metrics, nil
}
