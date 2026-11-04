package mta

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const stationMetadataURL = "https://data.ny.gov/resource/5f5g-n3cz.json?$limit=1000"

type StationMetadata struct {
	AccessibleIDs   map[string]struct{}
	BoroughByStopID map[string]string
}

type MetadataService struct {
	client   *http.Client
	mu       sync.Mutex
	value    StationMetadata
	loadedAt time.Time
}

func NewMetadataService() *MetadataService {
	return &MetadataService{client: &http.Client{Timeout: 6 * time.Second}}
}

func (s *MetadataService) Get(ctx context.Context) StationMetadata {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loadedAt.IsZero() && time.Since(s.loadedAt) < 6*time.Hour {
		return s.value
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, stationMetadataURL, nil)
	if err != nil {
		return StationMetadata{}
	}
	response, err := s.client.Do(request)
	if err != nil {
		return StationMetadata{}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return StationMetadata{}
	}
	var rows []struct {
		GTFSStopIDs string `json:"gtfs_stop_ids"`
		ADA         string `json:"ada"`
		Borough     string `json:"borough"`
	}
	if json.NewDecoder(response.Body).Decode(&rows) != nil {
		return StationMetadata{}
	}
	boroughs := map[string]string{"B": "Brooklyn", "Bk": "Brooklyn", "Bx": "Bronx", "M": "Manhattan", "Q": "Queens", "SI": "Staten Island"}
	value := StationMetadata{AccessibleIDs: map[string]struct{}{}, BoroughByStopID: map[string]string{}}
	for _, row := range rows {
		for _, id := range strings.Split(row.GTFSStopIDs, ";") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if row.ADA == "1" {
				value.AccessibleIDs[id] = struct{}{}
			}
			borough := boroughs[row.Borough]
			if borough == "" {
				borough = row.Borough
			}
			if borough != "" {
				value.BoroughByStopID[id] = borough
			}
		}
	}
	s.value, s.loadedAt = value, time.Now()
	return value
}
