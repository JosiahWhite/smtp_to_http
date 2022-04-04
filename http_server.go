package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"context"
)

type HTTPServer struct {
	listenAddress string
	secretToken   string
	messageStore  *MessageStore

	httpMux    *http.ServeMux
	httpServer *http.Server
}

func NewHTTPServer(listenAddress, secretToken string, messageStore *MessageStore) *HTTPServer {
	ret := &HTTPServer{
		listenAddress: listenAddress,
		secretToken:   secretToken,
		messageStore:  messageStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/fetchMessages", ret.fetchMessagesHandler)
	mux.HandleFunc("/clearMessages", ret.clearMessagesHandler)

	ret.httpMux = mux

	return ret
}

func (hs *HTTPServer) Run() error {
	l, err := net.Listen("tcp", hs.listenAddress)
	if err != nil {
		return fmt.Errorf("failed to start http server. Got: %s", err)
	}

	srv := &http.Server{
		Handler: hs.httpMux,
	}

	go func() {
		if err := srv.Serve(l); err != nil {
			log.Println("HTTP server returned error:", err)
			return
		}
	}()

	hs.httpServer = srv

	return nil
}

func (hs *HTTPServer) Stop() {
	// wait up to 30 seconds for the server to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hs.httpServer.Shutdown(ctx)
}

func (hs *HTTPServer) fetchMessagesHandler(w http.ResponseWriter, r *http.Request) {
	emailKeys, ok := r.URL.Query()["email"]
	if !ok || len(emailKeys) < 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Missing email parameter",
		})
		return
	}

	email := strings.ToLower(emailKeys[0])
	messages := hs.messageStore.FetchMessages(email)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"messages": messages,
	})
}

func (hs *HTTPServer) clearMessagesHandler(w http.ResponseWriter, r *http.Request) {
	emailKeys, ok := r.URL.Query()["email"]

	// if no email provided, return 400
	if !ok || len(emailKeys) < 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Missing email parameter",
		})
		return
	}

	email := strings.ToLower(emailKeys[0])
	hs.messageStore.RemoveMessages(email)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
