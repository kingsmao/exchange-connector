package futures_coin

import (
	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/schema"
)

// FuturesCoinExchange bundles REST and WS for MEXC Coin-margined Futures.
type FuturesCoinExchange struct {
	rest *FuturesCoinREST
	ws   *FuturesCoinWS
}

func NewFuturesCoinExchange(c *cache.MemoryCache) *FuturesCoinExchange {
	return &FuturesCoinExchange{
		rest: NewFuturesCoinREST(),
		ws:   NewFuturesCoinWS(c),
	}
}

func (f *FuturesCoinExchange) Name() schema.ExchangeName   { return schema.MEXC }
func (f *FuturesCoinExchange) Market() schema.MarketType   { return schema.FUTURESCOIN }
func (f *FuturesCoinExchange) REST() interfaces.RESTClient { return f.rest }
func (f *FuturesCoinExchange) WS() interfaces.WSConnector  { return f.ws }
