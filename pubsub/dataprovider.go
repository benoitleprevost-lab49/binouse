package pubsub

import (
	"log"
	"os"

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
	//TODO: ADD a mutex here
	if _, ok := p.subscriptions[symbol]; !ok {
		p.subscriptions[symbol] = make(chan market.Price)
	}
	return p.subscriptions[symbol]
}

func (p *PriceDispatcher) Unsubscribe(symbol string) bool {
	//TODO: ADD a mutex here
	if ch, ok := p.subscriptions[symbol]; ok {
		close(ch)
		delete(p.subscriptions, symbol)
		return true
	} else {
		return false
	}
}

func (p *PriceDispatcher) Start(sig <-chan os.Signal) {
	for {
		select {
		case <-sig:
			log.Println("Stoping dispatcher: the context is done")
			return
		case <-p.priceProducer.Done():
			log.Println("Stoping dispatcher: the price producer is done")
			return
		case price := <-p.priceProducer.Out():
			if ch, ok := p.subscriptions[price.Symbol]; ok {
				log.Println("Found subscription ", price.Symbol, " for price ", price.Price)
				ch <- price
			} else {
				log.Println("No subscription found for ", price.Symbol)
			}
		}
	}
}
