package futures_coin

import (
	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/schema"
)

// FuturesCoinExchange bundles REST and WS for Binance Coin-margined Futures.
type FuturesCoinExchange struct {
	rest *FuturesCoinREST
	ws   *FuturesCoinWS
}

func NewFuturesCoinExchange(c *cache.MemoryCache) *FuturesCoinExchange {
	subs := cache.NewSubscriptionManager()
	rest := NewFuturesCoinREST()
	return &FuturesCoinExchange{
		rest: rest,
		ws:   NewFuturesCoinWS(c, subs, rest),
	}
}

func (f *FuturesCoinExchange) Name() schema.ExchangeName   { return schema.BINANCE }
func (f *FuturesCoinExchange) Market() schema.MarketType   { return schema.FUTURESCOIN }
func (f *FuturesCoinExchange) REST() interfaces.RESTClient { return f.rest }
func (f *FuturesCoinExchange) WS() interfaces.WSConnector  { return f.ws }
