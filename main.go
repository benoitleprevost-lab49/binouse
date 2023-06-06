package main

import (
	"log"

	"github.com/benoitleprevost-lab49/binouse/market"
	"github.com/benoitleprevost-lab49/binouse/pubsub"
	"github.com/benoitleprevost-lab49/binouse/web"
)

var (
	apiKey    = "qF7DtXoU19fCrNpm7BWR4UbhAxpL37l5UvKQVpZbdZcaotf9CQMQNmhmQFgHXUwz"
	secretKey = "JjNzFYhVI8BZzrTNyqy0nOdBiGcmGks0IvAZ6K0QEzcsTydsKoQYkBpMIOcjwfKa"
)

func main() {
	log.Println("Starting server")
	// Get an exchange
	exchange := market.NewBinanceExchange(apiKey, secretKey)

	// Create a price producer
	errorer := func(err error) {
		log.Fatalln(err)
	}
	producer := exchange.ProduceAllPrice(errorer)

	// Create a price dispatcher
	dispatcher := pubsub.NewPriceDispatcher(producer)

	// Create an SSE server
	server := web.NewSseServer(dispatcher)
	server.Dummy("/dummy")
	server.Price("/price")
	server.Start()

}
