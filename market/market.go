package market

type Exchange interface {
	Symbols() ([]string, error)
	ProducePrice(symbols []string, errorHandler ErrorHandler) *PriceProducer
	ProduceAllPrice(errorHandler ErrorHandler) *PriceProducer
}

type Price struct {
	Symbol string
	Price  float64
	Time   int64
}

type PriceHandler func(x Price)

type ErrorHandler func(x error)

type PriceProducer interface {
	Close()
	Done() <-chan struct{}
	// Ctx() context.Context
	Out() <-chan Price
}
