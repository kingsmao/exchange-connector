package spot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const (
	okxBaseURL         = "https://www.okx.com"
	apiV5MarketCandles = "/api/v5/market/candles"
	apiV5MarketBooks   = "/api/v5/market/books"
)

type SpotREST struct{ http *resty.Client }

func NewSpotREST() *SpotREST {
	c := resty.New().SetBaseURL(okxBaseURL).SetTimeout(10 * time.Second)
	return &SpotREST{http: c}
}

func okxSymbol(sym string) string {
	// BTCUSDT -> BTC-USDT
	if strings.Contains(sym, "-") {
		return strings.ToUpper(sym)
	}
	if len(sym) > 4 {
		base := strings.ToUpper(sym[:len(sym)-4])
		quote := strings.ToUpper(sym[len(sym)-4:])
		if quote == "USDT" || quote == "USDC" || quote == "USDD" {
			return base + "-" + quote
		}
	}
	// fallback insert dash before last 3
	if len(sym) > 3 {
		return strings.ToUpper(sym[:len(sym)-3]) + "-" + strings.ToUpper(sym[len(sym)-3:])
	}
	return strings.ToUpper(sym)
}

func (o *SpotREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	var resp struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}
	r, err := o.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"instId": symbol,
		"bar":    string(interval),
		"limit":  fmt.Sprintf("%d", limit),
	}).Get(apiV5MarketCandles)
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("OKX Spot Kline 原始响应: %s", rawResponse)

	out := make([]schema.Kline, 0, len(resp.Data))
	for _, row := range resp.Data {
		if len(row) < 6 {
			continue
		}
		ts, _ := strconv.ParseInt(row[0], 10, 64)
		o, _ := decimal.NewFromString(row[1])
		h, _ := decimal.NewFromString(row[2])
		l, _ := decimal.NewFromString(row[3])
		c, _ := decimal.NewFromString(row[4])
		v, _ := decimal.NewFromString(row[5])
		qv, _ := decimal.NewFromString(row[6])
		out = append(out, schema.Kline{
			Exchange:    schema.OKX,
			Market:      schema.SPOT,
			Symbol:      symbol,
			Interval:    interval,
			OpenTime:    time.UnixMilli(ts),
			CloseTime:   time.UnixMilli(ts),
			Open:        o,
			High:        h,
			Low:         l,
			Close:       c,
			Volume:      v,
			QuoteVolume: qv,
			IsFinal:     true,
		})
	}
	return out, nil
}

func (o *SpotREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Ts   string     `json:"ts"`
		} `json:"data"`
	}
	r, err := o.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"instId": symbol,
		"sz":     fmt.Sprintf("%d", limit),
	}).Get(apiV5MarketBooks)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("OKX Spot Depth 原始响应: %s", rawResponse)

	if len(resp.Data) == 0 {
		return schema.Depth{}, errors.New("no depth data")
	}

	data := resp.Data[0]
	convert := func(levels [][]string) []schema.PriceLevel {
		out := make([]schema.PriceLevel, 0, len(levels))
		for _, level := range levels {
			if len(level) >= 2 {
				price, _ := decimal.NewFromString(level[0])
				quantity, _ := decimal.NewFromString(level[1])
				out = append(out, schema.PriceLevel{
					Price:    price,
					Quantity: quantity,
				})
			}
		}
		return out
	}

	ts, _ := strconv.ParseInt(data.Ts, 10, 64)
	return schema.Depth{
		Exchange:     schema.OKX,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         convert(data.Bids),
		Asks:         convert(data.Asks),
		UpdatedAt:    time.UnixMilli(ts),
		LastUpdateId: data.Ts,
	}, nil
}

func (o *SpotREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现OKX交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.OKX,
		Market:     schema.SPOT,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
