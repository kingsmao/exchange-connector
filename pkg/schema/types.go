package schema

import (
	"time"

	"github.com/shopspring/decimal"
)

// ExchangeName defines supported exchange.
type ExchangeName string

const (
	BINANCE ExchangeName = "binance"
	OKX     ExchangeName = "okx"
	BYBIT   ExchangeName = "bybit"
	GATE    ExchangeName = "gate"
	MEXC    ExchangeName = "mexc"
)

// MarketType categorizes market segments.
type MarketType string

const (
	SPOT        MarketType = "spot"
	FUTURESUSDT MarketType = "futures_usdt" // USDT-margined
	FUTURESCOIN MarketType = "futures_coin" // coin-margined
)

// Interval for Kline/candles.
type Interval string

const (
	Interval1m  Interval = "1m"
	Interval3m  Interval = "3m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval4h  Interval = "4h"
	Interval1d  Interval = "1d"
)

// PriceLevel represents a single order book level.
type PriceLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

// Depth represents order book snapshot.
type Depth struct {
	Exchange     ExchangeName `json:"exchange"`
	Market       MarketType   `json:"market"`
	Symbol       string       `json:"symbol"`
	Bids         []PriceLevel `json:"bids"` // 买盘,由大到小排序
	Asks         []PriceLevel `json:"asks"` // 卖盘,由小到大排序
	UpdatedAt    time.Time    `json:"updatedAt"`
	LastUpdateId string       `json:"rawVersion,omitempty"`
}

// Ticker represents the latest price.
type Ticker struct {
	Exchange  ExchangeName    `json:"exchange"`
	Market    MarketType      `json:"market"`
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Volume    decimal.Decimal `json:"volume,omitempty"`
	QuoteVol  decimal.Decimal `json:"quoteVolume,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Kline represents a normalized candle.
type Kline struct {
	Exchange    ExchangeName    `json:"exchange"`
	Market      MarketType      `json:"market"`
	Symbol      string          `json:"symbol"`
	Interval    Interval        `json:"interval"`
	OpenTime    time.Time       `json:"openTime"`
	CloseTime   time.Time       `json:"closeTime"`
	Open        decimal.Decimal `json:"open"`
	High        decimal.Decimal `json:"high"`
	Low         decimal.Decimal `json:"low"`
	Close       decimal.Decimal `json:"close"`
	Volume      decimal.Decimal `json:"volume"`
	QuoteVolume decimal.Decimal `json:"quoteVolume"`
	TradeNum    int64           `json:"tradeNum"`
	IsFinal     bool            `json:"isFinal"`
	EventTime   time.Time       `json:"-"`
	AdaptVolume decimal.Decimal `json:"-"`
}

// OrderSide defines the side of an order.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"  // 买单
	OrderSideSell OrderSide = "sell" // 卖单
)

// OrderType defines the type of an order.
type OrderType string

const (
	OrderTypeMarket OrderType = "market" // 市价单
	OrderTypeLimit  OrderType = "limit"  // 限价单
)

// OrderStatus defines the status of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"   // 待处理
	OrderStatusOpen      OrderStatus = "open"      // 已开仓
	OrderStatusFilled    OrderStatus = "filled"    // 已成交
	OrderStatusPartially OrderStatus = "partially" // 部分成交
	OrderStatusCanceled  OrderStatus = "canceled"  // 已取消
	OrderStatusRejected  OrderStatus = "rejected"  // 已拒绝
)

// Order represents a trading order.
type Order struct {
	Exchange        ExchangeName    `json:"exchange"`             // 交易所
	Market          MarketType      `json:"market"`               // 市场类型
	Symbol          string          `json:"symbol"`               // 交易对
	OrderID         string          `json:"orderId"`              // 订单ID
	ClientOrderID   string          `json:"clientOrderId"`        // 客户端订单ID
	Side            OrderSide       `json:"side"`                 // 订单方向
	Type            OrderType       `json:"type"`                 // 订单类型
	Status          OrderStatus     `json:"status"`               // 订单状态
	Price           decimal.Decimal `json:"price"`                // 价格
	Quantity        decimal.Decimal `json:"quantity"`             // 数量
	FilledQty       decimal.Decimal `json:"filledQty"`            // 已成交数量
	RemainingQty    decimal.Decimal `json:"remainingQty"`         // 剩余数量
	QuoteQty        decimal.Decimal `json:"quoteQty"`             // 报价数量
	FilledQuoteQty  decimal.Decimal `json:"filledQuoteQty"`       // 已成交报价数量
	Commission      decimal.Decimal `json:"commission"`           // 手续费
	CommissionAsset string          `json:"commissionAsset"`      // 手续费资产
	CreatedAt       time.Time       `json:"createdAt"`            // 创建时间
	UpdatedAt       time.Time       `json:"updatedAt"`            // 更新时间
	TimeInForce     string          `json:"timeInForce"`          // 有效期类型
	StopPrice       decimal.Decimal `json:"stopPrice,omitempty"`  // 止损价格
	IcebergQty      decimal.Decimal `json:"icebergQty,omitempty"` // 冰山数量
}

// Trade represents a completed trade.
type Trade struct {
	Exchange        ExchangeName    `json:"exchange"`        // 交易所
	Market          MarketType      `json:"market"`          // 市场类型
	Symbol          string          `json:"symbol"`          // 交易对
	TradeID         string          `json:"tradeId"`         // 成交ID
	OrderID         string          `json:"orderId"`         // 订单ID
	ClientOrderID   string          `json:"clientOrderId"`   // 客户端订单ID
	Side            OrderSide       `json:"side"`            // 成交方向
	Type            OrderType       `json:"type"`            // 订单类型
	Price           decimal.Decimal `json:"price"`           // 成交价格
	Quantity        decimal.Decimal `json:"quantity"`        // 成交数量
	QuoteQty        decimal.Decimal `json:"quoteQty"`        // 成交金额
	Commission      decimal.Decimal `json:"commission"`      // 手续费
	CommissionAsset string          `json:"commissionAsset"` // 手续费资产
	Timestamp       time.Time       `json:"timestamp"`       // 成交时间
	IsMaker         bool            `json:"isMaker"`         // 是否为挂单方
	Fee             decimal.Decimal `json:"fee"`             // 手续费（兼容字段）
	FeeAsset        string          `json:"feeAsset"`        // 手续费资产（兼容字段）
}

// ExchangeInfo 表示交易所的交易规则信息
type ExchangeInfo struct {
	Exchange   ExchangeName `json:"exchange"`   // 交易所名称
	Market     MarketType   `json:"market"`     // 市场类型
	Symbols    []Symbol     `json:"symbols"`    // 支持的交易对列表
	UpdatedAt  time.Time    `json:"updatedAt"`  // 更新时间
	ServerTime time.Time    `json:"serverTime"` // 服务器时间
	RateLimits []RateLimit  `json:"rateLimits"` // 接口限流规则
	Timezone   string       `json:"timezone"`   // 时区
}

// RateLimit 表示接口限流规则
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"` // 限流类型：REQUEST_WEIGHT, ORDERS, RAW_REQUESTS
	Interval      string `json:"interval"`      // 时间间隔：SECOND, MINUTE, DAY
	IntervalNum   int    `json:"intervalNum"`   // 间隔数量
	Limit         int    `json:"limit"`         // 限制数量
}

// Filter 表示交易对的过滤器规则
type Filter struct {
	FilterType  string `json:"filterType"`  // 过滤器类型
	MinPrice    string `json:"minPrice"`    // 最小价格
	MaxPrice    string `json:"maxPrice"`    // 最大价格
	TickSize    string `json:"tickSize"`    // 价格步长
	MinQty      string `json:"minQty"`      // 最小数量
	MaxQty      string `json:"maxQty"`      // 最大数量
	StepSize    string `json:"stepSize"`    // 数量步长
	MinNotional string `json:"minNotional"` // 最小名义价值
	MaxNotional string `json:"maxNotional"` // 最大名义价值
}

// SymbolStatus 定义交易对状态
type SymbolStatus string

const (
	SymbolStatusTrading      SymbolStatus = "TRADING"       // 正常交易
	SymbolStatusHalt         SymbolStatus = "HALT"          // 暂停交易
	SymbolStatusBreak        SymbolStatus = "BREAK"         // 交易暂停
	SymbolStatusAuctionMatch SymbolStatus = "AUCTION_MATCH" // 集合竞价
)
