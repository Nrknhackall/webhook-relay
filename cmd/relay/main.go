package main

import (
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "log"
 "net/http"
 "os"
 "sync"
 "time"
 "github.com/google/uuid"
)

type Endpoint struct { ID string `json:"id"`; URL string `json:"url"`; Secret string `json:"-"`; Active bool `json:"active"` }
type Delivery struct { ID string `json:"id"`; EndpointID string `json:"endpointId"`; Event string `json:"event"`; Status string `json:"status"`; Attempts int `json:"attempts"`; LastError string `json:"lastError,omitempty"` }
type Relay struct { mu sync.Mutex; endpoints map[string]Endpoint; deliveries map[string]Delivery; idempotency map[string]string }

func newRelay() *Relay { return &Relay{endpoints: map[string]Endpoint{}, deliveries: map[string]Delivery{}, idempotency: map[string]string{}} }
func (r *Relay) endpointsHandler(w http.ResponseWriter, req *http.Request) { w.Header().Set("Content-Type", "application/json"); r.mu.Lock(); defer r.mu.Unlock(); if req.Method == http.MethodPost { var input struct{ URL, Secret string }; if json.NewDecoder(req.Body).Decode(&input) != nil || input.URL == "" { http.Error(w, "invalid endpoint", 400); return }; e := Endpoint{ID: uuid.NewString(), URL: input.URL, Secret: input.Secret, Active: true}; r.endpoints[e.ID] = e; w.WriteHeader(201); json.NewEncoder(w).Encode(e); return }; out := make([]Endpoint, 0, len(r.endpoints)); for _, e := range r.endpoints { out = append(out, e) }; json.NewEncoder(w).Encode(out) }
func (r *Relay) messagesHandler(w http.ResponseWriter, req *http.Request) { w.Header().Set("Content-Type", "application/json"); var input struct{ EndpointID, Event string; Payload json.RawMessage }; if json.NewDecoder(req.Body).Decode(&input) != nil { http.Error(w, "invalid message", 400); return }; key := req.Header.Get("Idempotency-Key"); r.mu.Lock(); defer r.mu.Unlock(); if key != "" { if id := r.idempotency[key]; id != "" { json.NewEncoder(w).Encode(r.deliveries[id]); return } }; if _, ok := r.endpoints[input.EndpointID]; !ok { http.Error(w, "endpoint not found", 404); return }; d := Delivery{ID: uuid.NewString(), EndpointID: input.EndpointID, Event: input.Event, Status: "PENDING"}; r.deliveries[d.ID] = d; if key != "" { r.idempotency[key] = d.ID }; w.WriteHeader(202); json.NewEncoder(w).Encode(d) }
func (r *Relay) deliveriesHandler(w http.ResponseWriter, _ *http.Request) { r.mu.Lock(); defer r.mu.Unlock(); w.Header().Set("Content-Type", "application/json"); out := make([]Delivery, 0, len(r.deliveries)); for _, d := range r.deliveries { out = append(out, d) }; json.NewEncoder(w).Encode(out) }
func (r *Relay) replayHandler(w http.ResponseWriter, req *http.Request) { id := req.URL.Path[len("/v1/deliveries/"):]; if len(id) < len("/replay") { http.NotFound(w, req); return }; id = id[:len(id)-len("/replay")]; r.mu.Lock(); defer r.mu.Unlock(); d, ok := r.deliveries[id]; if !ok { http.NotFound(w, req); return }; d.Status = "PENDING"; d.Attempts = 0; r.deliveries[id] = d; json.NewEncoder(w).Encode(d) }
func sign(secret string, body []byte) string { mac := hmac.New(sha256.New, []byte(secret)); mac.Write(body); return hex.EncodeToString(mac.Sum(nil)) }
func retryDelay(attempt int) time.Duration { if attempt < 1 { attempt = 1 }; if attempt > 6 { attempt = 6 }; return time.Duration(1<<attempt) * time.Second }
func main() { relay := newRelay(); mux := http.NewServeMux(); mux.HandleFunc("/health", func(w http.ResponseWriter,_ *http.Request){ json.NewEncoder(w).Encode(map[string]string{"status":"ok"}) }); mux.HandleFunc("/v1/endpoints", relay.endpointsHandler); mux.HandleFunc("/v1/messages", relay.messagesHandler); mux.HandleFunc("/v1/deliveries", relay.deliveriesHandler); mux.HandleFunc("/v1/deliveries/", relay.replayHandler); log.Printf("webhook-relay listening on %s", os.Getenv("PORT")); log.Fatal(http.ListenAndServe(":"+env("PORT","4201"), mux)) }
func env(k, fallback string) string { if v:=os.Getenv(k);v!=""{return v};return fallback }
