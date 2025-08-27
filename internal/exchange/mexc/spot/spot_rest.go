package spot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"

	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"
)

const (
	mexcBaseURL      = "https://api.mexc.com"
	apiV3TickerPrice = "/api/v3/ticker/price"
	apiV3Kline       = "/api/v3/kline"
	apiV3Depth       = "/api/v3/depth"
)

type SpotREST struct{ http *resty.Client }

func NewSpotREST() *SpotREST {
	return &SpotREST{http: resty.New().SetBaseURL(mexcBaseURL).SetTimeout(10 * time.Second)}
}

func (m *SpotREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	var resp struct {
		Symbol    string `json:"symbol"`
		Price     string `json:"price"`
		Volume    string `json:"volume"`
		QuoteVol  string `json:"quoteVolume"`
		Timestamp int64  `json:"time"`
	}
	r, err := m.http.R().SetContext(ctx).SetResult(&resp).SetQueryParam("symbol", symbol).Get(apiV3TickerPrice)
	if err != nil {
		return schema.Ticker{}, err
	}
	if r.IsError() {
		return schema.Ticker{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("MEXC Spot Ticker 原始响应: %s", rawResponse)

	price, _ := decimal.NewFromString(resp.Price)
	volume, _ := decimal.NewFromString(resp.Volume)
	quoteVolume, _ := decimal.NewFromString(resp.QuoteVol)

	return schema.Ticker{
		Exchange:  schema.MEXC,
		Market:    schema.SPOT,
		Symbol:    resp.Symbol,
		Price:     price,
		Volume:    volume,
		QuoteVol:  quoteVolume,
		Timestamp: time.UnixMilli(resp.Timestamp),
	}, nil
}

func (s *SpotREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	var resp [][]interface{}
	r, err := s.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{"symbol": symbol, "interval": string(interval), "limit": fmt.Sprintf("%d", limit)}).Get(apiV3Kline)
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, errors.New(r.Status())
	}
	out := make([]schema.Kline, 0, len(resp))
	for _, row := range resp {
		if len(row) < 11 {
			continue
		}
		ts := int64(row[0].(float64))
		o, _ := decimal.NewFromString(row[1].(string))
		h, _ := decimal.NewFromString(row[2].(string))
		l, _ := decimal.NewFromString(row[3].(string))
		c, _ := decimal.NewFromString(row[4].(string))
		v, _ := decimal.NewFromString(row[5].(string))
		out = append(out, schema.Kline{Exchange: schema.MEXC, Market: schema.SPOT, Symbol: symbol, Interval: interval, OpenTime: time.UnixMilli(ts), CloseTime: time.UnixMilli(ts), Open: o, High: h, Low: l, Close: c, Volume: v, IsFinal: true})
	}
	return out, nil
}

func (m *SpotREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		LastUpdateId int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}
	r, err := m.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"symbol": symbol,
		"limit":  fmt.Sprintf("%d", limit),
	}).Get(apiV3Depth)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("MEXC Spot Depth 原始响应: %s", rawResponse)

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

	return schema.Depth{
		Exchange:     schema.MEXC,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         convert(resp.Bids),
		Asks:         convert(resp.Asks),
		UpdatedAt:    time.Now(),
		LastUpdateId: fmt.Sprintf("%d", resp.LastUpdateId),
	}, nil
}

func (m *SpotREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现MEXC交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.MEXC,
		Market:     schema.SPOT,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
