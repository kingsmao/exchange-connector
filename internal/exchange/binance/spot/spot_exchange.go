package spot

import (
	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// SpotExchange bundles REST and WS for Binance Spot.
type SpotExchange struct {
	rest *SpotREST
	ws   *SpotWS
}

func NewSpotExchange(c *cache.MemoryCache) *SpotExchange {
	return &SpotExchange{
		rest: NewSpotREST(),
		ws:   NewSpotWS(c, NewSpotREST()),
	}
}

func (s *SpotExchange) Name() schema.ExchangeName   { return schema.BINANCE }
func (s *SpotExchange) Market() schema.MarketType   { return schema.SPOT }
func (s *SpotExchange) REST() interfaces.RESTClient { return s.rest }
func (s *SpotExchange) WS() interfaces.WSConnector  { return s.ws }
