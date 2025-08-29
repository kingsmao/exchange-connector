package futures_usdt

import (
	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// FuturesUSDTExchange bundles REST and WS for Binance USDT-margined Futures.
type FuturesUSDTExchange struct {
	rest *FuturesUSDTREST
	ws   *FuturesUSDTWS
}

func NewFuturesUSDTExchange(c *cache.MemoryCache) *FuturesUSDTExchange {
	subs := cache.NewSubscriptionManager()
	rest := NewFuturesUSDTREST()
	return &FuturesUSDTExchange{
		rest: rest,
		ws:   NewFuturesUSDTWS(c, subs, rest),
	}
}

func (f *FuturesUSDTExchange) Name() schema.ExchangeName   { return schema.BINANCE }
func (f *FuturesUSDTExchange) Market() schema.MarketType   { return schema.FUTURESUSDT }
func (f *FuturesUSDTExchange) REST() interfaces.RESTClient { return f.rest }
func (f *FuturesUSDTExchange) WS() interfaces.WSConnector  { return f.ws }
