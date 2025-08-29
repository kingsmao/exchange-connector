package futures_usdt

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
	binanceFuturesUSDTBaseURL = "https://fapi.binance.com"
	apiV1Ticker24hr           = "/fapi/v1/ticker/24hr"
	apiV1Kline                = "/fapi/v1/klines"
	apiV1Depth                = "/fapi/v1/depth"
	apiV1ExchangeInfo         = "/fapi/v1/exchangeInfo"
)

// FuturesUSDTREST implements RESTClient for Binance USDT-margined Futures.
type FuturesUSDTREST struct {
	http *resty.Client
}

func NewFuturesUSDTREST() *FuturesUSDTREST {
	return &FuturesUSDTREST{
		http: resty.New().SetBaseURL(binanceFuturesUSDTBaseURL),
	}
}

func (f *FuturesUSDTREST) GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error) {
	var resp struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}
	r, err := f.http.R().SetContext(ctx).SetResult(&resp).SetQueryParams(map[string]string{
		"symbol": symbol,
		"limit":  fmt.Sprintf("%d", limit),
	}).Get(apiV1Depth)
	if err != nil {
		return schema.Depth{}, err
	}
	if r.IsError() {
		return schema.Depth{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Futures USDT Depth 原始响应: %s", rawResponse)
	convert := func(levels [][]string) []schema.PriceLevel {
		out := make([]schema.PriceLevel, 0, len(levels))
		for _, lv := range levels {
			if len(lv) < 2 {
				continue
			}
			p, _ := decimal.NewFromString(lv[0])
			q, _ := decimal.NewFromString(lv[1])
			out = append(out, schema.PriceLevel{Price: p, Quantity: q})
		}
		return out
	}
	return schema.Depth{
		Exchange:     schema.BINANCE,
		Market:       schema.FUTURESUSDT,
		Symbol:       symbol,
		Bids:         convert(resp.Bids),
		Asks:         convert(resp.Asks),
		UpdatedAt:    time.Now(),
		LastUpdateId: fmt.Sprintf("%d", resp.LastUpdateID),
	}, nil
}

func (f *FuturesUSDTREST) GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error) {
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
			Symbol                string   `json:"symbol"`
			Status                string   `json:"status"`
			MaintMarginPercent    string   `json:"maintMarginPercent"`
			RequiredMarginPercent string   `json:"requiredMarginPercent"`
			BaseAsset             string   `json:"baseAsset"`
			QuoteAsset            string   `json:"quoteAsset"`
			MarginAsset           string   `json:"marginAsset"`
			PricePrecision        int      `json:"pricePrecision"`
			QuantityPrecision     int      `json:"quantityPrecision"`
			BaseAssetPrecision    int      `json:"baseAssetPrecision"`
			QuotePrecision        int      `json:"quotePrecision"`
			UnderlyingType        string   `json:"underlyingType"`
			UnderlyingSubType     []string `json:"underlyingSubType"`
			SettlePlan            int      `json:"settlePlan"`
			TriggerProtect        string   `json:"triggerProtect"`
			OrderTypes            []string `json:"orderTypes"`
			TimeInForce           []string `json:"timeInForce"`
			Filters               []struct {
				FilterType     string `json:"filterType"`
				MinPrice       string `json:"minPrice,omitempty"`
				MaxPrice       string `json:"maxPrice,omitempty"`
				TickSize       string `json:"tickSize,omitempty"`
				MinQty         string `json:"minQty,omitempty"`
				MaxQty         string `json:"maxQty,omitempty"`
				StepSize       string `json:"stepSize,omitempty"`
				MinNotional    string `json:"minNotional,omitempty"`
				MaxNotional    string `json:"maxNotional,omitempty"`
				Limit          int    `json:"limit,omitempty"`
				MultiplierUp   string `json:"multiplierUp,omitempty"`
				MultiplierDown string `json:"multiplierDown,omitempty"`
			} `json:"filters"`
		} `json:"symbols"`
	}

	r, err := f.http.R().SetContext(ctx).SetResult(&resp).Get(apiV1ExchangeInfo)
	if err != nil {
		return schema.ExchangeInfo{}, err
	}
	if r.IsError() {
		return schema.ExchangeInfo{}, errors.New(r.Status())
	}

	// 保存原始响应结果用于调试
	rawResponse := string(r.Body())
	logger.Debug("Binance Futures USDT ExchangeInfo 原始响应长度: %d bytes", len(rawResponse))

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
		// 只处理状态为TRADING的交易对
		if s.Status != "TRADING" {
			continue
		}

		symbol := schema.Symbol{
			Symbol:       s.Symbol,
			Base:         s.BaseAsset,
			Quote:        s.QuoteAsset,
			Margin:       s.MarginAsset, // USDT合约的保证金资产
			ExchangeName: schema.BINANCE,
			MarketType:   schema.FUTURESUSDT,

			// 精度信息
			QuantityPrecision: s.QuantityPrecision,
			PricePrecision:    s.PricePrecision,
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
				// 期货合约使用NOTIONAL而不是MIN_NOTIONAL
				if symbol.MinNotional == "" {
					symbol.MinNotional = filter.MinNotional
				}
			case "MARKET_LOT_SIZE":
				// 市价单数量过滤器，可能需要单独处理
			case "MAX_NUM_ORDERS":
				// 最大订单数量过滤器
			case "MAX_NUM_ALGO_ORDERS":
				// 最大算法订单数量过滤器
			case "PERCENT_PRICE":
				// 价格百分比过滤器
			}
		}

		symbols = append(symbols, symbol)
	}

	return schema.ExchangeInfo{
		Exchange:   schema.BINANCE,
		Market:     schema.FUTURESUSDT,
		Symbols:    symbols,
		UpdatedAt:  time.Now(),
		ServerTime: time.UnixMilli(resp.ServerTime),
		RateLimits: rateLimits,
		Timezone:   resp.Timezone,
	}, nil
}
