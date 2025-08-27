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
	spotBaseURL = "https://api.binance.com"

	// API endpoints
	apiV3Depth        = "/api/v3/depth"
	apiV3ExchangeInfo = "/api/v3/exchangeInfo"
)

// SpotREST implements RESTClient for Binance Spot.
type SpotREST struct {
	http *resty.Client
}

func NewSpotREST() *SpotREST {
	c := resty.New().SetBaseURL(spotBaseURL).SetTimeout(10 * time.Second)
	return &SpotREST{http: c}
}

func (s *SpotREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}
	r, err := s.http.R().SetContext(ctx).
		SetQueryParams(map[string]string{
			"symbol": symbol,
			"limit":  fmt.Sprintf("%d", limit),
		}).
		SetResult(&resp).
		Get(apiV3Depth)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Spot Depth 原始响应: %s", rawResponse)
	convert := func(levels [][]string) []schema.PriceLevel {
		out := make([]schema.PriceLevel, 0, len(levels))
		for _, lv := range levels {
			if len(lv) < 2 {
				continue
			}
			// 使用16位精度解析价格和数量，并去除末尾的0位
			p, _ := decimal.NewFromString(lv[0])
			q, _ := decimal.NewFromString(lv[1])
			out = append(out, schema.PriceLevel{Price: p, Quantity: q})
		}
		return out
	}

	return schema.Depth{
		Exchange:     schema.BINANCE,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         convert(resp.Bids),
		Asks:         convert(resp.Asks),
		UpdatedAt:    time.Now(),
		LastUpdateId: fmt.Sprintf("%d", resp.LastUpdateID),
	}, nil
}

func (s *SpotREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
	var resp struct {
		Timezone   string `json:"timezone"`
		ServerTime int64  `json:"serverTime"`
		RateLimits []struct {
			RateLimitType string `json:"rateLimitType"`
			Interval      string `json:"interval"`
			IntervalNum   int    `json:"intervalNum"`
			Limit         int    `json:"limit"`
		} `json:"rateLimits"`
		Symbols []struct {
			Symbol                     string   `json:"symbol"`
			Status                     string   `json:"status"`
			BaseAsset                  string   `json:"baseAsset"`
			BaseAssetPrecision         int      `json:"baseAssetPrecision"`
			QuoteAsset                 string   `json:"quoteAsset"`
			QuoteAssetPrecision        int      `json:"quoteAssetPrecision"`
			OrderTypes                 []string `json:"orderTypes"`
			IcebergAllowed             bool     `json:"icebergAllowed"`
			OcoAllowed                 bool     `json:"ocoAllowed"`
			QuoteOrderQtyMarketAllowed bool     `json:"quoteOrderQtyMarketAllowed"`
			AllowTrailingStop          bool     `json:"allowTrailingStop"`
			CancelReplaceAllowed       bool     `json:"cancelReplaceAllowed"`
			IsSpotTradingAllowed       bool     `json:"isSpotTradingAllowed"`
			IsMarginTradingAllowed     bool     `json:"isMarginTradingAllowed"`
			Filters                    []struct {
				FilterType  string `json:"filterType"`
				MinPrice    string `json:"minPrice,omitempty"`
				MaxPrice    string `json:"maxPrice,omitempty"`
				TickSize    string `json:"tickSize,omitempty"`
				MinQty      string `json:"minQty,omitempty"`
				MaxQty      string `json:"maxQty,omitempty"`
				StepSize    string `json:"stepSize,omitempty"`
				MinNotional string `json:"minNotional,omitempty"`
				MaxNotional string `json:"maxNotional,omitempty"`
			} `json:"filters"`
			Permissions []string `json:"permissions"`
		} `json:"symbols"`
	}

	r, err := s.http.R().SetContext(ctx).SetResult(&resp).Get(apiV3ExchangeInfo)
	if err != nil {
		return schema.ExchangeInfo{}, err
	}
	if r.IsError() {
		return schema.ExchangeInfo{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Spot ExchangeInfo 原始响应长度: %d bytes", len(rawResponse))

	// 转换限流规则
	rateLimits := make([]schema.RateLimit, len(resp.RateLimits))
	for i, rl := range resp.RateLimits {
		rateLimits[i] = schema.RateLimit{
			RateLimitType: rl.RateLimitType,
			Interval:      rl.Interval,
			IntervalNum:   rl.IntervalNum,
			Limit:         rl.Limit,
		}
	}

	// 转换交易对信息
	symbols := make([]schema.Symbol, 0, len(resp.Symbols))
	for _, s := range resp.Symbols {
		// 只处理现货交易且状态为TRADING的交易对
		if s.Status != "TRADING" || !s.IsSpotTradingAllowed {
			continue
		}

		symbol := schema.Symbol{
			Symbol:       s.Symbol,
			Base:         s.BaseAsset,
			Quote:        s.QuoteAsset,
			Margin:       "", // 现货无保证金
			ExchangeName: schema.BINANCE,
			MarketType:   schema.SPOT,

			// 初始化精度信息
			QuantityPrecision: s.BaseAssetPrecision,
			PricePrecision:    s.QuoteAssetPrecision,
		}

		// 解析过滤器信息
		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "LOT_SIZE":
				symbol.MinQuantity = filter.MinQty
				symbol.MaxQuantity = filter.MaxQty
			case "MIN_NOTIONAL":
				symbol.MinNotional = filter.MinNotional
			case "NOTIONAL":
				symbol.MinNotional = filter.MinNotional
			}
		}

		symbols = append(symbols, symbol)
	}

	return schema.ExchangeInfo{
		Exchange:   schema.BINANCE,
		Market:     schema.SPOT,
		Symbols:    symbols,
		UpdatedAt:  time.Now(),
		ServerTime: time.UnixMilli(resp.ServerTime),
		RateLimits: rateLimits,
		Timezone:   resp.Timezone,
	}, nil
}
