package spot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const (
	gateBaseURL         = "https://api.gateio.ws/api/v4"
	apiSpotTickers      = "/spot/tickers"
	apiSpotCandlesticks = "/spot/candlesticks"
	apiSpotOrderBook    = "/spot/order_book"
)

type SpotREST struct{ http *resty.Client }

func NewSpotREST() *SpotREST {
	return &SpotREST{http: resty.New().SetBaseURL(gateBaseURL).SetTimeout(10 * time.Second)}
}

func (s *SpotREST) GetTicker(ctx context.Context, symbol string) (schema.Ticker, error) {
	var resp struct {
		CurrencyPair     string `json:"currency_pair"`
		Last             string `json:"last"`
		LowestAsk        string `json:"lowest_ask"`
		HighestBid       string `json:"highest_bid"`
		ChangePercentage string `json:"change_percentage"`
		BaseVolume       string `json:"base_volume"`
		QuoteVolume      string `json:"quote_volume"`
		High24h          string `json:"high_24h"`
		Low24h           string `json:"low_24h"`
	}
	r, err := s.http.R().SetContext(ctx).SetResult(&resp).SetQueryParam("currency_pair", symbol).Get(apiSpotTickers)
	if err != nil {
		return schema.Ticker{}, err
	}
	if r.IsError() {
		return schema.Ticker{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Gate Spot Ticker 原始响应: %s", rawResponse)

	price, _ := decimal.NewFromString(resp.Last)
	volume, _ := decimal.NewFromString(resp.BaseVolume)
	quoteVolume, _ := decimal.NewFromString(resp.QuoteVolume)

	return schema.Ticker{
		Exchange:  schema.GATE,
		Market:    schema.SPOT,
		Symbol:    resp.CurrencyPair,
		Price:     price,
		Volume:    volume,
		QuoteVol:  quoteVolume,
		Timestamp: time.Now(),
	}, nil
}

func (s *SpotREST) GetKline(ctx context.Context, symbol string, interval schema.Interval, limit int) ([]schema.Kline, error) {
	var resp [][]string
	r, err := s.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{"currency_pair": symbol, "interval": string(interval), "limit": fmt.Sprintf("%d", limit)}).Get(apiSpotCandlesticks)
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, errors.New(r.Status())
	}
	out := make([]schema.Kline, 0, len(resp))
	for _, row := range resp {
		if len(row) < 6 {
			continue
		}
		ts, _ := strconv.ParseInt(row[0], 10, 64)
		o, _ := decimal.NewFromString(row[5])
		h, _ := decimal.NewFromString(row[3])
		l, _ := decimal.NewFromString(row[4])
		c, _ := decimal.NewFromString(row[2])
		v, _ := decimal.NewFromString(row[1])
		out = append(out, schema.Kline{Exchange: schema.GATE, Market: schema.SPOT, Symbol: symbol, Interval: interval, OpenTime: time.Unix(ts, 0), CloseTime: time.Unix(ts, 0), Open: o, High: h, Low: l, Close: c, Volume: v, IsFinal: true})
	}
	return out, nil
}

func (s *SpotREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		Id      int64      `json:"id"`
		Current int64      `json:"current"`
		Update  int64      `json:"update"`
		Asks    [][]string `json:"asks"`
		Bids    [][]string `json:"bids"`
	}
	r, err := s.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"currency_pair": symbol,
		"limit":         fmt.Sprintf("%d", limit),
	}).Get(apiSpotOrderBook)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Gate Spot Depth 原始响应: %s", rawResponse)

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
		Exchange:     schema.GATE,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         convert(resp.Bids),
		Asks:         convert(resp.Asks),
		UpdatedAt:    time.Unix(resp.Update, 0),
		LastUpdateId: fmt.Sprintf("%d", resp.Id),
	}, nil
}

func (s *SpotREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	// 暂未实现Gate交易规则信息获取
	return schema.ExchangeInfo{
		Exchange:   schema.GATE,
		Market:     schema.SPOT,
		Symbols:    []schema.Symbol{},
		UpdatedAt:  time.Now(),
		ServerTime: time.Now(),
		RateLimits: []schema.RateLimit{},
		Timezone:   "UTC",
	}, nil
}
