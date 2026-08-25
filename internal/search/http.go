package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type QueryResponse struct {
	Results []Result `json:"results"`
}

type Node struct {
	Index *Index
	WAL   *WAL
}

func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "documents": n.Index.Len()})
	})
	mux.HandleFunc("POST /v1/documents", n.handleDocument)
	mux.HandleFunc("GET /v1/search", n.handleSearch)
	return mux
}

func (n *Node) handleDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&doc); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(doc.ID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}
	if err := n.WAL.Append(doc); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "durability failure"})
		return
	}
	n.Index.Upsert(doc)
	writeJSON(w, http.StatusCreated, map[string]string{"id": doc.ID})
}

func (n *Node) handleSearch(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	writeJSON(w, http.StatusOK, QueryResponse{Results: n.Index.Search(r.URL.Query().Get("q"), limit)})
}

type ReplicaSet struct {
	Name     string
	Replicas []string
}

type Coordinator struct {
	Shards []ReplicaSet
	Client *http.Client
}

func NewCoordinator(shards []ReplicaSet) *Coordinator {
	return &Coordinator{Shards: shards, Client: &http.Client{Timeout: 2 * time.Second}}
}

// ShardFor applies rendezvous hashing, which minimizes movement when shards
// are added or removed without requiring a centralized hash ring.
func (c *Coordinator) ShardFor(key string) (ReplicaSet, error) {
	if len(c.Shards) == 0 {
		return ReplicaSet{}, errors.New("no shards configured")
	}
	selected := c.Shards[0]
	var best uint64
	for _, shard := range c.Shards {
		hash := fnv.New64a()
		_, _ = hash.Write([]byte(key + "\x00" + shard.Name))
		if score := hash.Sum64(); score >= best {
			best, selected = score, shard
		}
	}
	return selected, nil
}

func (c *Coordinator) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "shards": len(c.Shards)})
	})
	mux.HandleFunc("POST /v1/documents", c.handleDocument)
	mux.HandleFunc("GET /v1/search", c.handleSearch)
	return mux
}

func (c *Coordinator) handleDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&doc); err != nil || doc.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid document id and JSON body required"})
		return
	}
	shard, err := c.ShardFor(doc.ID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	payload, _ := json.Marshal(doc)
	type outcome struct{ err error }
	outcomes := make(chan outcome, len(shard.Replicas))
	for _, replica := range shard.Replicas {
		go func(baseURL string) {
			req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/documents", strings.NewReader(string(payload)))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				var response *http.Response
				response, err = c.Client.Do(req)
				if err == nil {
					defer response.Body.Close()
					if response.StatusCode/100 != 2 {
						err = fmt.Errorf("replica returned %s", response.Status)
					}
				}
			}
			outcomes <- outcome{err: err}
		}(replica)
	}
	successes := 0
	for range shard.Replicas {
		if (<-outcomes).err == nil {
			successes++
		}
	}
	quorum := len(shard.Replicas)/2 + 1
	if successes < quorum {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "replica quorum unavailable", "acknowledged": successes, "required": quorum})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": doc.ID, "shard": shard.Name, "replicas": successes})
}

func (c *Coordinator) handleSearch(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	type shardResult struct {
		results []Result
		err     error
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2500*time.Millisecond)
	defer cancel()
	responses := make(chan shardResult, len(c.Shards))
	for _, shard := range c.Shards {
		go func(set ReplicaSet) {
			var lastErr error
			for _, replica := range set.Replicas {
				requestURL := strings.TrimRight(replica, "/") + "/v1/search?q=" + url.QueryEscape(r.URL.Query().Get("q")) + "&limit=" + strconv.Itoa(limit)
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
				response, err := c.Client.Do(req)
				if err != nil {
					lastErr = err
					continue
				}
				var body QueryResponse
				err = json.NewDecoder(response.Body).Decode(&body)
				response.Body.Close()
				if err == nil && response.StatusCode/100 == 2 {
					responses <- shardResult{results: body.Results}
					return
				}
				lastErr = err
			}
			responses <- shardResult{err: lastErr}
		}(shard)
	}

	all := make([]Result, 0, len(c.Shards)*limit)
	failed := 0
	for range c.Shards {
		response := <-responses
		if response.err != nil {
			failed++
			continue
		}
		all = append(all, response.results...)
	}
	if failed == len(c.Shards) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "all shards unavailable"})
		return
	}
	sort.Slice(all, func(a, b int) bool { return all[a].Score > all[b].Score })
	if len(all) > limit {
		all = all[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": all, "failed_shards": failed})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
