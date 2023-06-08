package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/benoitleprevost-lab49/binouse/market"
	"github.com/benoitleprevost-lab49/binouse/pubsub"
	"github.com/benoitleprevost-lab49/binouse/web"
)

const apiKey = "qF7DtXoU19fCrNpm7BWR4UbhAxpL37l5UvKQVpZbdZcaotf9CQMQNmhmQFgHXUwz"
const secretKey = "JjNzFYhVI8BZzrTNyqy0nOdBiGcmGks0IvAZ6K0QEzcsTydsKoQYkBpMIOcjwfKa"

func main() {
	log.Println("Starting server")
	// global context for shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Get an exchange
	exchange := market.NewBinanceExchange(apiKey, secretKey)

	// Create a price producer
	errorer := func(err error) {
		// we could call cancel() maybe based on the error??
		log.Fatalln(err)
	}
	producer := exchange.ProduceAllPrice(errorer)

	// Create a price dispatcher
	dispatcher := pubsub.NewPriceDispatcher(producer)
	dispatcher.Start()

	// Create an SSE server

	server := web.NewSseServer(dispatcher)
	server.Dummy(ctx, "/dummy")
	server.Price(ctx, "/price")

	// shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	server.Start(ctx)

}
