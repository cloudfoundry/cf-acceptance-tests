package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Stats struct {
	// Locker  sync.Locker
	Latency []float64 `json:"latency"`
	sync.RWMutex
}

func (s *Stats) Add(latency float64) {
	s.Lock()
	defer s.Unlock()
	s.Latency = append(s.Latency, latency)
}

func (s *Stats) Clear() {
	s.Lock()
	defer s.Unlock()
	s.Latency = []float64{}
}

func (s *Stats) GetLatency() []float64 {
	s.RLock()
	defer s.RUnlock()
	return s.Latency
}

type StatsHandler struct {
	Stats *Stats
}

func (h *StatsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "DELETE" {
		h.Stats.Clear()
		return
	}

	respBytes, err := json.Marshal(h.Stats)
	if err != nil {
		panic(err)
	}
	resp.Write(respBytes)
	return
}
