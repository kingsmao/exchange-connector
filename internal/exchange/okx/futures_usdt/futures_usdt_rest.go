package futures_usdt

import (
	"context"
	"errors"
	"time"

	"github.com/go-resty/resty/v2"

	"exchange-connector/pkg/schema"
)

const okxFuturesUSDTBaseURL = "https://www.okx.com"

// FuturesUSDTREST implements RESTClient for OKX USDT-margined Futures.
type FuturesUSDTREST struct {
	http *resty.Client
}

func NewFuturesUSDTREST() *FuturesUSDTREST {
	return &FuturesUSDTREST{
		http: resty.New().SetBaseURL(okxFuturesUSDTBaseURL),
	}
}

func (f *FuturesUSDTREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	// TODO: Implement OKX futures USDT ticker
	return schema.Ticker{}, errors.New("not implemented")
}

func (f *FuturesUSDTREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	// TODO: Implement OKX futures USDT kline
	return nil, errors.New("not implemented")
}

func (f *FuturesUSDTREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	// TODO: Implement OKX futures USDT depth
	return schema.Depth{}, errors.New("not implemented")
}

func (f *FuturesUSDTREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现OKX USDT合约交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.OKX,
		Market:     schema.FUTURESUSDT,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
