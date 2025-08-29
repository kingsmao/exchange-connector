package futures_usdt

import (
	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// FuturesUSDTExchange bundles REST and WS for Gate USDT-margined Futures.
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

func (f *FuturesUSDTExchange) Name() schema.ExchangeName   { return schema.GATE }
func (f *FuturesUSDTExchange) Market() schema.MarketType   { return schema.FUTURESUSDT }
func (f *FuturesUSDTExchange) REST() interfaces.RESTClient { return f.rest }
func (f *FuturesUSDTExchange) WS() interfaces.WSConnector  { return f.ws }
