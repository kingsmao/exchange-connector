package spot

import (
	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/schema"
)

// SpotExchange bundles REST and WS for OKX Spot.
type SpotExchange struct {
	rest *SpotREST
	ws   *SpotWS
}

func NewSpotExchange(c *cache.MemoryCache) *SpotExchange {
	subs := cache.NewSubscriptionManager()
	return &SpotExchange{
		rest: NewSpotREST(),
		ws:   NewSpotWS(c, subs),
	}
}

func (s *SpotExchange) Name() schema.ExchangeName   { return schema.OKX }
func (s *SpotExchange) Market() schema.MarketType   { return schema.SPOT }
func (s *SpotExchange) REST() interfaces.RESTClient { return s.rest }
func (s *SpotExchange) WS() interfaces.WSConnector  { return s.ws }
