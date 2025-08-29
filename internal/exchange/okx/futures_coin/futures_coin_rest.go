package futures_coin

import (
	"context"
	"errors"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const OkxFuturesCoinBaseURL = "https://www.okx.com"

// FuturesCoinREST implements RESTClient for Okx Coin-margined Futures.
type FuturesCoinREST struct {
	http *resty.Client
}

func NewFuturesCoinREST() *FuturesCoinREST {
	return &FuturesCoinREST{
		http: resty.New().SetBaseURL(OkxFuturesCoinBaseURL),
	}
}

func (f *FuturesCoinREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	// TODO: Implement Okx futures coin ticker
	return schema.Ticker{}, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	// TODO: Implement Okx futures coin kline
	return nil, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	// TODO: Implement Okx futures coin depth
	return schema.Depth{}, errors.New("not implemented")
}

func (f *FuturesCoinREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现OKX币本位合约交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.OKX,
		Market:     schema.FUTURESCOIN,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
