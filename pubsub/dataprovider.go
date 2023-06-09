package pubsub

import (
	"context"
	"log"

	"github.com/benoitleprevost-lab49/binouse/market"
)

type PriceDispatcher struct {
	subscriptions map[string](chan market.Price)
	priceProducer market.PriceProducer
}

func NewPriceDispatcher(priceProducer market.PriceProducer) *PriceDispatcher {
	return &PriceDispatcher{
		subscriptions: make(map[string](chan market.Price)),
		priceProducer: priceProducer,
	}
}

func (p *PriceDispatcher) Subscribe(symbol string) <-chan market.Price {
	if _, ok := p.subscriptions[symbol]; !ok {
		p.subscriptions[symbol] = make(chan market.Price)
	}
	return p.subscriptions[symbol]
}

func (p *PriceDispatcher) Unsubscribe(symbol string) bool {
	if ch, ok := p.subscriptions[symbol]; ok {
		close(ch)
		delete(p.subscriptions, symbol)
		return true
	} else {
		return false
	}
}

func (p *PriceDispatcher) process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			log.Print(err)
			return
		case <-p.priceProducer.Done():
			log.Println("Price producer is done")
			return
		case price := <-p.priceProducer.Out():
			if _, ok := p.subscriptions[price.Symbol]; ok {
				log.Println("Found subscription ", price.Symbol, " for price ", price.Price)
				p.subscriptions[price.Symbol] <- price
			} else {
				log.Println("No subscription found for ", price.Symbol)
			}
		}
	}
}

func (p *PriceDispatcher) Start() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	go p.process(ctx)

	return ctx, cancel
}
