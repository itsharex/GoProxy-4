package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/proxy"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
)

type sseSnapshot struct {
	Status      proxy.Status               `json:"status"`
	Stats       stats.Stats                `json:"stats"`
	Connections []proxy.ConnectionSnapshot `json:"connections"`
}

func (a *WebApp) serveSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ctx := r.Context()

	fmt.Fprint(w, ":ok\n\n")
	flusher.Flush()

	var lastUp, lastDown int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			var snapshot sseSnapshot
			if a.server != nil {
				snapshot.Status = a.server.Status()
				snapshot.Stats = a.server.Stats()
				snapshot.Connections = a.server.ActiveConnections()
			}
			a.mu.Unlock()

			if lastUp > 0 || lastDown > 0 {
				snapshot.Stats.UploadRate = max64(0, snapshot.Stats.UploadBytes-lastUp)
				snapshot.Stats.DownloadRate = max64(0, snapshot.Stats.DownloadBytes-lastDown)
			}
			lastUp = snapshot.Stats.UploadBytes
			lastDown = snapshot.Stats.DownloadBytes

			data, err := json.Marshal(snapshot)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
