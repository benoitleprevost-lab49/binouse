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
	client := binance.NewClient(b.apiKey, b.secretKey)
	ei, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	symbols := make([]string, 0)
	for _, s := range ei.Symbols {
		if s.IsMarginTradingAllowed && s.IsSpotTradingAllowed {
			symbols = append(symbols, s.Symbol)
		}

	}
	return symbols, nil
}

func (b *BinanceExchange) ProduceAllPrice(errorHandler ErrorHandler) *BinanceProducer {
	start := time.Now()
	tokens, err := b.Symbols()
	if err != nil {
		errorHandler(err)
	}
	elapsed := time.Since(start)
	log.Println("Found", len(tokens), "Tokens in", elapsed)
	log.Println("Symbols:", tokens)
	return b.ProducePrice(tokens, errorHandler)
}

func (b *BinanceExchange) ProducePrice(symbols []string, errorHandler ErrorHandler) *BinanceProducer {
	outC := make(chan Price)
	tradeEventHandler := func(event *binance.WsAggTradeEvent) {
		log.Println("Receveived AggTradeEvent")
		price, err := strconv.ParseFloat(event.Price, 64)
		if err != nil {
			errorHandler(err)
		}
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
	b.stopC <- struct{}{}
	close(b.outC)
}

func (b *BinanceProducer) Done() <-chan struct{} {
	return b.doneC
}

func (b *BinanceProducer) Out() <-chan Price {
	return b.outC
}
