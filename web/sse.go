package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/benoitleprevost-lab49/binouse/pubsub"
	"github.com/gorilla/mux"
)

type SseServer struct {
	mux             *mux.Router
	priceDispatcher *pubsub.PriceDispatcher
}

func NewSseServer(priceDispatcher *pubsub.PriceDispatcher) *SseServer {
	return &SseServer{mux: mux.NewRouter(), priceDispatcher: priceDispatcher}
}

func (s *SseServer) Dummy(ctx context.Context, pattern string) {
	log.Println("Registering dummer handler for pattern: ", pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Fatalln("Streaming unsupported!")
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		timeout := time.After(30 * time.Second)
		id := 0
		for {
			select {
			case <-ctx.Done():
				log.Println("Context done")
				return
			case <-timeout:
				return
			default:
				time.Sleep(1 * time.Second)
				fmt.Fprintf(w, "data: %s\n\n", "Event "+fmt.Sprint(id))
				flusher.Flush()
				id += 1
			}
		}

	})
}

func (s *SseServer) Price(ctx context.Context, pattern string) {
	log.Println("Registering Price handler for pattern: ", pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher := w.(http.Flusher)
		timeout := time.After(30 * time.Second)

		symbol := r.URL.Query().Get("symbol")
		if symbol != "" {
			prices := s.priceDispatcher.Subscribe(symbol)
			for {
				select {
				case <-ctx.Done():
					log.Println("Context done")
					return
				case <-timeout:
					log.Println("Timeout")
					s.priceDispatcher.Unsubscribe(symbol)
					return
				case <-r.Context().Done():
					log.Println("Context done")
					s.priceDispatcher.Unsubscribe(symbol)
					return
				case price := <-prices:
					byt, _ := json.Marshal(price)
					fmt.Fprintf(w, "data: %s\n\n", string(byt))
					flusher.Flush()
				}
			}
		} else {
			fmt.Fprintf(w, "data: %s\n\n", "No symbol provided")
			flusher.Flush()
		}

	})
}

func (s *SseServer) Start(ctx context.Context) {
	log.Println("Starting server on port 8080 ...")
	// start the server
	srv := &http.Server{
		Addr: "0.0.0.0:8080",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      s.mux, // Pass our instance of gorilla/mux in.
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("server started")

	<-ctx.Done()

	log.Printf("server stopped")

	// gracefull shutdown sequence
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("server Shutdown Failed:%+s", err)
	}

	log.Printf("server exited properly")
}
