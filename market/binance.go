package market

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
)

type BinanceExchange struct {
	apiKey    string
	secretKey string
}

func NewBinanceExchange(apiKey, secretKey string) *BinanceExchange {
	return &BinanceExchange{apiKey: apiKey, secretKey: secretKey}
}

func (b *BinanceExchange) Symbols() ([]string, error) {
	// this client is not reusable, it should be created outside BinanceExchange and pass in the constructor
	client := binance.NewClient(b.apiKey, b.secretKey)
	ei, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	symbols := make([]string, 0) // you could set the cap here, because we know the max number of items
	for _, s := range ei.Symbols {
		if s.IsMarginTradingAllowed && s.IsSpotTradingAllowed {
			symbols = append(symbols, s.Symbol)
		}

	}
	return symbols, nil
}

// this method is unnecessary, you could do the timing and logging in Symbols() or from the call site
func (b *BinanceExchange) ProduceAllPrice(errorHandler ErrorHandler) *BinanceProducer {
	start := time.Now()
	tokens, err := b.Symbols()
	if err != nil {
		// you should return the error here, doesn't make sense to go forward if the tokens slice is empty
		// plus I prefer to wrap the errors, either with just extra information about the actual functionality or with a stack trace too
		errorHandler(err)
	}
	elapsed := time.Since(start)
	log.Println("Found", len(tokens), "Tokens in", elapsed)
	log.Println("Symbols:", tokens)
	return b.ProducePrice(tokens, errorHandler)
}

func (b *BinanceExchange) ProducePrice(symbols []string, errorHandler ErrorHandler) *BinanceProducer {
	outC := make(chan Price, 64)
	tradeEventHandler := func(event *binance.WsAggTradeEvent) {

		price, err := strconv.ParseFloat(event.Price, 64)
		if err != nil {
			errorHandler(err)
		}
		// log.Println("Receveived AggTradeEvent:", event.Symbol)
		outC <- Price{
			Symbol: event.Symbol,
			Price:  price,
			Time:   event.Time,
		}
	}

	binErroHandler := func(err error) {
		errorHandler(err)
	}

	doneC, stopC, err := binance.WsCombinedAggTradeServe(symbols, tradeEventHandler, binErroHandler)
	if err != nil {
		errorHandler(err)
	}

	return &BinanceProducer{
		doneC: doneC,
		stopC: stopC,
		outC:  outC,
	}
}

type BinanceProducer struct {
	doneC chan struct{}
	stopC chan struct{}
	outC  chan Price
}

func (b *BinanceProducer) Close() {
	// funny thing, this is not thread safe, but not because of your code, the binance library has a bug :)
	b.stopC <- struct{}{}
	// you should wait on the doneC channel before closing outC, otherwise the tradeEventHandler function can panic
	close(b.outC)
}

// we usually don't use getters in Go, just make them public in the struct
func (b *BinanceProducer) Done() <-chan struct{} {
	return b.doneC
}

func (b *BinanceProducer) Out() <-chan Price {
	return b.outC
}
