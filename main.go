package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/benoitleprevost-lab49/binouse/market"
	"github.com/benoitleprevost-lab49/binouse/pubsub"
	"github.com/benoitleprevost-lab49/binouse/web"
)

const apiKey = "qF7DtXoU19fCrNpm7BWR4UbhAxpL37l5UvKQVpZbdZcaotf9CQMQNmhmQFgHXUwz"
const secretKey = "JjNzFYhVI8BZzrTNyqy0nOdBiGcmGks0IvAZ6K0QEzcsTydsKoQYkBpMIOcjwfKa"

func main() {
	// one signal channel to know it's been cancelled
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// starting the server
	log.Println("Starting server")

	// Get an exchange
	exchange := market.NewBinanceExchange(apiKey, secretKey)

	// Create a price producer
	producer, err := exchange.ProduceAllPrice()
	if err != nil {
		log.Fatal("error creating price producer: ", err)
	}

	// Create a price dispatcher
	dispatcher := pubsub.NewPriceDispatcher(producer)
	go func() {
		dispatcher.Start(sig)
	}()

	// Create an SSE sseserver
	sseserver := web.NewSseServer(dispatcher)
	sseserver.Dummy(sig, "/dummy")
	sseserver.Price(sig, "/price")

	go func() {
		sseserver.Start()

	}()

	log.Println("Server started !!!")

	oscall := <-sig
	log.Printf("system call:%+v", oscall)
	close(sig)

	// shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	go func() {
		sseserver.Shutdown(ctx)
		cancel()
	}()

	<-ctx.Done()
	log.Println("Terminated !!!")
}
