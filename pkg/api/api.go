package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"github.com/weaveworks-experiments/loki/pkg/storage"
)

const (
	defaultWindowMS = 60 * 60 * 1000
)

func parseInt64(values url.Values, key string, def int64) (int64, error) {
	value := values.Get(key)
	if value == "" {
		return def, nil
	}

	intVal, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return intVal, nil
}

func Register(router *mux.Router, store *storage.SpanStore) {
	router.Handle("/api/v1/dependencies", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(struct{}{}); err != nil {
			log.Errorf("Error marshalling: %v", err)
		}
	}))

	router.Handle("/config.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(struct {
			DefaultLookback int `json:"defaultLookback"`
			QueryLimit      int `json:"queryLimit"`
		}{
			DefaultLookback: 3600000,
			QueryLimit:      10,
		}); err != nil {
			log.Errorf("Error marshalling config: %v", err)
		}
	}))

	router.Handle("/api/v1/services", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(store.Services()); err != nil {
			log.Errorf("Error marshalling: %v", err)
		}
	}))

	router.Handle("/api/v1/spans", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		serviceName := values.Get("serviceName")
		if serviceName == "" {
			http.Error(w, "serviceName required", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(store.SpanNames(serviceName)); err != nil {
			log.Errorf("Error marshalling: %v", err)
		}
	}))

	router.Handle("/api/v1/trace/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := fromIDStr(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		trace := store.Trace(id)
		if err := json.NewEncoder(w).Encode(SpansToWire(trace)); err != nil {
			log.Errorf("Error marshalling: %v", err)
		}
	}))

	router.Handle("/api/v1/traces", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nowMS := time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
		values := r.URL.Query()

		startTS, err := parseInt64(values, "startTS", nowMS-defaultWindowMS)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		endTS, err := parseInt64(values, "endTS", nowMS)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		serviceName := values.Get("serviceName")
		if serviceName == "" {
			http.Error(w, "serviceName required", http.StatusBadRequest)
			return
		}

		query := storage.Query{
			EndMS:       endTS,
			StartMS:     startTS,
			Limit:       10,
			ServiceName: serviceName,
			SpanName:    values.Get("spanName"),
		}
		traces := store.Traces(query)
		if err := json.NewEncoder(w).Encode(TracesToWire(traces)); err != nil {
			log.Errorf("Error marshalling: %v", err)
		}
	}))
}
