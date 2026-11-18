package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	store *Store
	admin string // admin token for panel API
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func randomToken() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) checkAdmin(r *http.Request) bool {
	// Header first, then ?token= for convenience from CLI.
	t := r.Header.Get("X-Admin-Token")
	if t == "" {
		t = r.URL.Query().Get("admin")
	}
	return t != "" && t == s.admin
}

func decodeBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, 4<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Panel static assets are embedded in assets.go.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(indexHTML)
	})

	// Admin API: list / create / update / delete node sets.
	mux.HandleFunc("/api/sets", func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAdmin(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin token required"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]interface{}{"sets": s.store.List(), "formats": formats})
		case http.MethodPost, http.MethodPut:
			var set NodeSet
			if err := decodeBody(r, &set); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			if len(set.Nodes) == 0 {
				writeJSON(w, 400, map[string]string{"error": "at least one node is required"})
				return
			}
			if err := s.store.Upsert(&set); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]interface{}{"set": set})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/sets/", func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAdmin(r) {
			writeJSON(w, 401, map[string]string{"error": "admin token required"})
			return
		}
		token := strings.TrimPrefix(r.URL.Path, "/api/sets/")
		if token == "" || strings.Contains(token, "/") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			if err := s.store.Delete(token); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]string{"ok": "deleted"})
			return
		}
		set, ok := s.store.Get(token)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, map[string]interface{}{"set": set})
	})

	// Public subscription endpoint: /sub/{token}/{format}
	mux.HandleFunc("/sub/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 3 || parts[0] != "sub" {
			http.NotFound(w, r)
			return
		}
		token, formatID := parts[1], parts[2]
		set, ok := s.store.Get(token)
		if !ok {
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}
		body, contentType, ok := generateSubscription(formatID, set.Nodes, set.Name)
		if !ok {
			http.Error(w, "unknown subscription format", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "inline; filename=\""+set.Name+"-"+formatID+"\"")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=0; total=0; expire=0")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		io.WriteString(w, body)
	})

	// List of all subscription URLs for a token (admin only).
	mux.HandleFunc("/api/urls", func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAdmin(r) {
			writeJSON(w, 401, map[string]string{"error": "admin token required"})
			return
		}
		token := r.URL.Query().Get("token")
		if _, ok := s.store.Get(token); !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		base := scheme + "://" + r.Host
		urls := make([]map[string]string, 0, len(formats))
		for _, f := range formats {
			urls = append(urls, map[string]string{
				"id": f.ID, "name": f.Name, "url": base + "/sub/" + token + "/" + f.Path,
			})
		}
		writeJSON(w, 200, map[string]interface{}{"urls": urls})
	})

	return mux
}

func serve(addr, dataFile, adminToken string) error {
	store, err := OpenStore(dataFile)
	if err != nil {
		return err
	}
	srv := &Server{store: store, admin: adminToken}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		srv.routes().ServeHTTP(w, r)
	})
	return http.ListenAndServe(addr, h)
}

var _ = strconv.Itoa
