package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/benoitleprevost-lab49/binouse/pubsub"
)

type SseServer struct {
	mux             *http.ServeMux
	priceDispatcher *pubsub.PriceDispatcher
}

func NewSseServer(priceDispatcher *pubsub.PriceDispatcher) *SseServer {
	return &SseServer{mux: http.NewServeMux(), priceDispatcher: priceDispatcher}
}

func (s *SseServer) Dummy(pattern string) {
	log.Println("Registering dummer handler for pattern: ", pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher := w.(http.Flusher)
		timeout := time.After(30 * time.Second)
		id := 0
		for {
			select {
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

func (s *SseServer) Price(pattern string) {
	log.Println("Registering Price handler for pattern: ", pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher := w.(http.Flusher) // if "w" is not a Flusher, it can panic
		timeout := time.After(30 * time.Second)

		symbol := r.URL.Query().Get("symbol")
		if symbol != "" {
			prices := s.priceDispatcher.Subscribe(symbol)
			for {
				// this can be an endless loop, you are not handling the case when another thread closing the same "prices" channel
				// if another goroutine close the channel and this one is not timed out,
				// then you will send the default value rapidly and maybe endlessly to the client in the "<-prices" case
				select {
				case <-timeout:
					log.Println("Timeout")
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

func (s *SseServer) Start() {
	log.Println("Starting server on port 8080 ...")
	log.Fatal(http.ListenAndServe(":8080", s.mux)) // you will log a fatal message even if it was closed correctly :)
}
