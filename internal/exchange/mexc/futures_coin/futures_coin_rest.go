package futures_coin

import (
	"context"
	"errors"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const MexcFuturesCoinBaseURL = "https://www.mexc.com"

// FuturesCoinREST implements RESTClient for Mexc Coin-margined Futures.
type FuturesCoinREST struct {
	http *resty.Client
}

func NewFuturesCoinREST() *FuturesCoinREST {
	return &FuturesCoinREST{
		http: resty.New().SetBaseURL(MexcFuturesCoinBaseURL),
	}
}

func (f *FuturesCoinREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	// TODO: Implement Mexc futures coin ticker
	return schema.Ticker{}, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	// TODO: Implement Mexc futures coin kline
	return nil, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	// TODO: Implement Mexc futures coin depth
	return schema.Depth{}, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现MEXC币本位合约交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.MEXC,
		Market:     schema.FUTURESCOIN,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
