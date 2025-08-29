package futures_coin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const (
	binanceFuturesCoinBaseURL = "https://dapi.binance.com"
	apiV1Ticker24hr           = "/dapi/v1/ticker/24hr"
	apiV1Kline                = "/dapi/v1/klines"
	apiV1Depth                = "/dapi/v1/depth"
	apiV1ExchangeInfo         = "/dapi/v1/exchangeInfo"
)

// FuturesCoinREST implements RESTClient for Binance Coin-margined Futures.
type FuturesCoinREST struct {
	http *resty.Client
}

func NewFuturesCoinREST() *FuturesCoinREST {
	return &FuturesCoinREST{
		http: resty.New().SetBaseURL(binanceFuturesCoinBaseURL),
	}
}

func (f *FuturesCoinREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		LastUpdateId int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}
	r, err := f.http.R().SetContext(ctx).SetQueryParams(map[string]string{
		"symbol": symbol,
		"limit":  fmt.Sprintf("%d", limit),
	}).SetResult(&resp).Get(apiV1Depth)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Futures Coin Depth 原始响应: %s", rawResponse)
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
		Exchange:     schema.BINANCE,
		Market:       schema.FUTURESCOIN,
		Symbol:       symbol,
		Bids:         convert(resp.Bids),
		Asks:         convert(resp.Asks),
		UpdatedAt:    time.Now(),
		LastUpdateId: fmt.Sprintf("%d", resp.LastUpdateId),
	}, nil
}

func (f *FuturesCoinREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
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
			Symbol            string `json:"symbol"`
			ContractStatus    string `json:"contractStatus"`
			BaseAsset         string `json:"baseAsset"`
			QuoteAsset        string `json:"quoteAsset"`
			PricePrecision    int    `json:"pricePrecision"`
			QuantityPrecision int    `json:"quantityPrecision"`
			MinQuantity       string `json:"minQty"`
			MinNotional       string `json:"minNotional"`
			Filters           []struct {
				FilterType  string      `json:"filterType"`
				MinQty      interface{} `json:"minQty,omitempty"`
				MinNotional interface{} `json:"minNotional,omitempty"`
			} `json:"filters"`
		} `json:"symbols"`
	}

	r, err := f.http.R().SetContext(ctx).SetResult(&resp).Get(apiV1ExchangeInfo)
	if err != nil {
		return schema.ExchangeInfo{}, fmt.Errorf("获取交易规则失败: %v", err)
	}
	if r.IsError() {
		return schema.ExchangeInfo{}, fmt.Errorf("API错误: %s", r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Futures Coin ExchangeInfo 原始响应: %s", rawResponse)

	// 转换交易对信息
	symbols := make([]schema.Symbol, 0, len(resp.Symbols))
	for _, s := range resp.Symbols {
		// 只缓存可交易的币对
		if s.ContractStatus != "TRADING" {
			continue
		}

		// 从filters中提取minQty和minNotional
		var minQty, minNotional string
		for _, filter := range s.Filters {
			if filter.FilterType == "LOT_SIZE" {
				if minQtyVal, ok := filter.MinQty.(string); ok {
					minQty = minQtyVal
				}
			}
			if filter.FilterType == "MIN_NOTIONAL" {
				if minNotionalVal, ok := filter.MinNotional.(string); ok {
					minNotional = minNotionalVal
				}
			}
		}

		// 如果没有找到minNotional，使用默认值
		if minNotional == "" {
			minNotional = "0"
		}

		symbols = append(symbols, schema.Symbol{
			Symbol:            s.Symbol,
			Base:              s.BaseAsset,
			Quote:             s.QuoteAsset,
			ExchangeName:      schema.BINANCE,
			MarketType:        schema.FUTURESCOIN,
			QuantityPrecision: s.QuantityPrecision,
			PricePrecision:    s.PricePrecision,
			MinQuantity:       minQty,
			MinNotional:       minNotional,
		})
	}

	// 转换限流信息
	rateLimits := make([]schema.RateLimit, 0, len(resp.RateLimits))
	for _, rl := range resp.RateLimits {
		rateLimits = append(rateLimits, schema.RateLimit{
			RateLimitType: rl.RateLimitType,
			Interval:      rl.Interval,
			IntervalNum:   rl.IntervalNum,
			Limit:         rl.Limit,
		})
	}

	return schema.ExchangeInfo{
		Exchange:   schema.BINANCE,
		Market:     schema.FUTURESCOIN,
		Symbols:    symbols,
		UpdatedAt:  time.Now(),
		ServerTime: time.UnixMilli(resp.ServerTime),
		RateLimits: rateLimits,
		Timezone:   resp.Timezone,
	}, nil
}
