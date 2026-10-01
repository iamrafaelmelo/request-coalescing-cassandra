package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iamrafaelmelo/requests-coalescing/internal/cassandra"
	"github.com/iamrafaelmelo/requests-coalescing/internal/coalescing"
)

type MessageResponse struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Host    string `json:"host"`
}

func main() {
	repository, err := cassandra.NewRepository()
	if err != nil {
		log.Fatalf("failed to connect to Cassandra: %v", err)
	}
	defer repository.Close()

	service := coalescing.NewService(repository)

	http.HandleFunc("/messages/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/messages/")

		if id == "" {
			http.Error(w, "message id is required", http.StatusBadRequest)
			return
		}

		content, host, err := service.GetMessage(id)
		if err != nil {
			http.Error(w, "message not found", http.StatusNotFound)
			return
		}

		response := MessageResponse{
			ID:      id,
			Content: content,
			Host:    host,
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		fmt.Fprintf(
			w,
			"cassandra_queries_total %d\n",
			repository.QueryCount(),
		)
	})

	log.Println("API listening on :8080")

	server := &http.Server{
		Addr:           ":8080",
		Handler:        http.DefaultServeMux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 8 << 10,
	}

	log.Println("API listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
