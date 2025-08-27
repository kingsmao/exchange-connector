package spot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"

	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"
)

const (
	bybitBaseURL         = "https://api.bybit.com"
	apiV5MarketTickers   = "/v5/market/tickers"
	apiV5MarketKline     = "/v5/market/kline"
	apiV5MarketOrderbook = "/v5/market/orderbook"
)

type SpotREST struct{ http *resty.Client }

func NewSpotREST() *SpotREST {
	return &SpotREST{http: resty.New().SetBaseURL(bybitBaseURL).SetTimeout(10 * time.Second)}
}

func (b *SpotREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol        string `json:"symbol"`
				LastPrice     string `json:"lastPrice"`
				PrevPrice24h  string `json:"prevPrice24h"`
				Price24hPcnt  string `json:"price24hPcnt"`
				HighPrice24h  string `json:"highPrice24h"`
				LowPrice24h   string `json:"lowPrice24h"`
				Turnover24h   string `json:"turnover24h"`
				Volume24h     string `json:"volume24h"`
				UsdIndexPrice string `json:"usdIndexPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	r, err := b.http.R().SetContext(ctx).SetResult(&resp).SetQueryParam("category", "spot").SetQueryParam("symbol", symbol).Get(apiV5MarketTickers)
	if err != nil {
		return schema.Ticker{}, err
	}
	if r.IsError() {
		return schema.Ticker{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Bybit Spot Ticker 原始响应: %s", rawResponse)

	if len(resp.Result.List) == 0 {
		return schema.Ticker{}, errors.New("no ticker data")
	}

	data := resp.Result.List[0]
	price, _ := decimal.NewFromString(data.LastPrice)
	volume, _ := decimal.NewFromString(data.Volume24h)
	quoteVolume, _ := decimal.NewFromString(data.Turnover24h)

	return schema.Ticker{
		Exchange:  schema.BYBIT,
		Market:    schema.SPOT,
		Symbol:    data.Symbol,
		Price:     price,
		Volume:    volume,
		QuoteVol:  quoteVolume,
		Timestamp: time.Now(),
	}, nil
}

func intervalBybit(iv schema.Interval) string {
	m := map[schema.Interval]string{schema.Interval1m: "1", schema.Interval3m: "3", schema.Interval5m: "5", schema.Interval15m: "15", schema.Interval30m: "30", schema.Interval1h: "60", schema.Interval4h: "240", schema.Interval1d: "D"}
	if v, ok := m[iv]; ok {
		return v
	}
	return "1"
}

func (b *SpotREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string     `json:"category"`
			Symbol   string     `json:"symbol"`
			List     [][]string `json:"list"`
		} `json:"result"`
	}
	r, err := b.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"category": "spot",
		"symbol":   symbol,
		"interval": string(interval),
		"limit":    fmt.Sprintf("%d", limit),
	}).Get(apiV5MarketKline)
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Bybit Spot Kline 原始响应: %s", rawResponse)

	out := make([]schema.Kline, 0, len(resp.Result.List))
	for _, row := range resp.Result.List {
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
			Exchange:    schema.BYBIT,
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

func (b *SpotREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string     `json:"category"`
			Symbol   string     `json:"symbol"`
			Bids     [][]string `json:"bids"`
			Asks     [][]string `json:"asks"`
		} `json:"result"`
	}
	r, err := b.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"category": "spot",
		"symbol":   symbol,
		"limit":    fmt.Sprintf("%d", limit),
	}).Get(apiV5MarketOrderbook)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Bybit Spot Depth 原始响应: %s", rawResponse)

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
		Exchange:     schema.BYBIT,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         convert(resp.Result.Bids),
		Asks:         convert(resp.Result.Asks),
		UpdatedAt:    time.Now(),
		LastUpdateId: fmt.Sprintf("%d", time.Now().UnixMilli()),
	}, nil
}

func (b *SpotREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现Bybit交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.BYBIT,
		Market:     schema.SPOT,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
