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
	client    *binance.Client
}

func NewBinanceExchange(apiKey, secretKey string) *BinanceExchange {
	client := binance.NewClient(apiKey, secretKey)
	return &BinanceExchange{apiKey: apiKey, secretKey: secretKey, client: client}
}

func (b *BinanceExchange) Symbols() ([]string, error) {
	start := time.Now()

	ei, err := b.client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	symbols := make([]string, 0, len(ei.Symbols)) // we don't know yet how many symbols we will be tradeable
	for _, s := range ei.Symbols {
		if s.IsMarginTradingAllowed && s.IsSpotTradingAllowed {
			symbols = append(symbols, s.Symbol)
		}
	}

	elapsed := time.Since(start)

	log.Println("Found", len(symbols), "Tokens in", elapsed)
	log.Println("Symbols:", symbols)

	return symbols, nil
}

func (b *BinanceExchange) ProduceAllPrice() (*BinanceProducer, error) {
	tokens, err := b.Symbols()
	if err != nil {
		return nil, err
	}

	producer, err := b.ProducePrice(tokens)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func (b *BinanceExchange) ProducePrice(symbols []string) (*BinanceProducer, error) {
	outC := make(chan Price)

	tradeEventHandler := func(event *binance.WsAggTradeEvent) {
		price, err := strconv.ParseFloat(event.Price, 64)
		if err != nil {
			log.Fatalln("Error parsing price:", err)
			return
		}
		// log.Println("Receveived AggTradeEvent:", event.Symbol)
		outC <- Price{
			Symbol: event.Symbol,
			Price:  price,
			Time:   event.Time,
		}
	}

	binErroHandler := func(err error) {
		log.Fatalln("Error:", err)
	}

	doneC, stopC, err := binance.WsCombinedAggTradeServe(symbols, tradeEventHandler, binErroHandler)
	if err != nil {
		return nil, err
	}

	return &BinanceProducer{
		doneC: doneC,
		stopC: stopC,
		outC:  outC,
	}, nil
}

type BinanceProducer struct {
	doneC chan struct{}
	stopC chan struct{}
	outC  chan Price
}

// This getters are there to implement the PriceProducer interface

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
