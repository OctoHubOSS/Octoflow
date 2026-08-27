package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type checkRecord struct {
	CheckedAt  string `json:"checked_at"`
	Reachable  bool   `json:"reachable"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Database   bool   `json:"database"`
	Discord    bool   `json:"discord"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type healthBody struct {
	Database bool `json:"database"`
	Discord  bool `json:"discord"`
}

var fileMu sync.Mutex

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	targetURL := getenv("TARGET_URL", "https://v2.gitlogs.xyz")
	dataFile := getenv("DATA_FILE", "/data/status-history.ndjson")
	addr := getenv("LISTEN_ADDR", ":8090")

	interval, err := time.ParseDuration(getenv("CHECK_INTERVAL", "5m"))
	if err != nil {
		log.Fatalf("invalid CHECK_INTERVAL: %v", err)
	}

	maxLines, err := strconv.Atoi(getenv("MAX_LINES", "30000"))
	if err != nil {
		log.Fatalf("invalid MAX_LINES: %v", err)
	}

	timeout, err := time.ParseDuration(getenv("CHECK_TIMEOUT", "10s"))
	if err != nil {
		log.Fatalf("invalid CHECK_TIMEOUT: %v", err)
	}

	client := &http.Client{Timeout: timeout}

	go runChecks(client, targetURL, dataFile, maxLines, interval)

	mux := http.NewServeMux()
	mux.HandleFunc("/status-history.ndjson", func(w http.ResponseWriter, r *http.Request) {
		serveHistoryFile(w, dataFile)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Printf("status-checker listening on %s, checking %s every %s", addr, targetURL, interval)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func runChecks(client *http.Client, targetURL, dataFile string, maxLines int, interval time.Duration) {
	check(client, targetURL, dataFile, maxLines)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		check(client, targetURL, dataFile, maxLines)
	}
}

func check(client *http.Client, targetURL, dataFile string, maxLines int) {
	rec := checkRecord{CheckedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z")}

	start := time.Now()
	resp, err := client.Get(targetURL + "/api/health")
	rec.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		rec.Reachable = false
		rec.Error = err.Error()
	} else {
		defer resp.Body.Close()
		rec.Reachable = true
		rec.HTTPStatus = resp.StatusCode

		if body, readErr := io.ReadAll(resp.Body); readErr == nil {
			var hb healthBody
			if json.Unmarshal(body, &hb) == nil {
				rec.Database = hb.Database
				rec.Discord = hb.Discord
			}
		}
	}

	appendRecord(dataFile, rec, maxLines)
	log.Printf("checked %s: reachable=%v database=%v discord=%v latency=%dms", targetURL, rec.Reachable, rec.Database, rec.Discord, rec.LatencyMs)
}

func appendRecord(dataFile string, rec checkRecord, maxLines int) {
	fileMu.Lock()
	defer fileMu.Unlock()

	line, err := json.Marshal(rec)
	if err != nil {
		log.Printf("could not marshal record: %v", err)
		return
	}

	f, err := os.OpenFile(dataFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("could not open data file: %v", err)
		return
	}
	_, writeErr := f.Write(append(line, '\n'))
	f.Close()
	if writeErr != nil {
		log.Printf("could not write record: %v", writeErr)
	}

	trimIfNeeded(dataFile, maxLines)
}

func trimIfNeeded(dataFile string, maxLines int) {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}

	lines := splitLines(data)
	if len(lines) <= maxLines {
		return
	}

	trimmed := lines[len(lines)-maxLines:]
	out := make([]byte, 0, len(data))
	for _, l := range trimmed {
		out = append(out, l...)
		out = append(out, '\n')
	}

	if err := os.WriteFile(dataFile, out, 0644); err != nil {
		log.Printf("could not trim data file: %v", err)
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func serveHistoryFile(w http.ResponseWriter, dataFile string) {
	fileMu.Lock()
	defer fileMu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	f, err := os.Open(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	io.Copy(w, f)
}
