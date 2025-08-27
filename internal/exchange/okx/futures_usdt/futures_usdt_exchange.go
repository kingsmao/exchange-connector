package futures_usdt

import (
	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/schema"
)

// FuturesUSDTExchange bundles REST and WS for OKX USDT-margined Futures.
type FuturesUSDTExchange struct {
	rest *FuturesUSDTREST
	ws   *FuturesUSDTWS
}

func NewFuturesUSDTExchange(c *cache.MemoryCache) *FuturesUSDTExchange {
	return &FuturesUSDTExchange{
		rest: NewFuturesUSDTREST(),
		ws:   NewFuturesUSDTWS(c),
	}
}

func (f *FuturesUSDTExchange) Name() schema.ExchangeName   { return schema.OKX }
func (f *FuturesUSDTExchange) Market() schema.MarketType   { return schema.FUTURESUSDT }
func (f *FuturesUSDTExchange) REST() interfaces.RESTClient { return f.rest }
func (f *FuturesUSDTExchange) WS() interfaces.WSConnector  { return f.ws }
