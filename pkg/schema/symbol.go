package schema

import (
	"fmt"
	"strings"
)

// Symbol 表示一个完整的币对信息
type Symbol struct {
	Symbol       string       `json:"symbol"`       // 交易所格式的币对符号
	Base         string       `json:"base"`         // 基础币种
	Quote        string       `json:"quote"`        // 计价币种
	Margin       string       `json:"margin"`       // 保证金币种（期货时存在）
	ExchangeName ExchangeName `json:"exchangeName"` // 交易所名称
	MarketType   MarketType   `json:"marketType"`   // 市场类型

	// 交易规则信息
	QuantityPrecision int    `json:"quantityPrecision"` // 数量精度（小数位数）
	PricePrecision    int    `json:"pricePrecision"`    // 价格精度（小数位数）
	MinQuantity       string `json:"minQuantity"`       // 最小下单数量
	MinNotional       string `json:"minNotional"`       // 最小下单金额
	MaxQuantity       string `json:"maxQuantity"`       // 最大下单数量（可选）
}

// NewSymbol 创建一个新的Symbol实例
func NewSymbol(symbol, base, quote, margin string, exchangeName ExchangeName, marketType MarketType) *Symbol {
	return &Symbol{
		Symbol:       symbol,
		Base:         strings.ToUpper(base),
		Quote:        strings.ToUpper(quote),
		Margin:       strings.ToUpper(margin),
		ExchangeName: exchangeName,
		MarketType:   marketType,
	}
}

// NewSymbolFromString 从字符串创建Symbol实例（用于兼容性）
func NewSymbolFromString(symbol, base, quote, margin, exchangeName, marketType string) *Symbol {
	return &Symbol{
		Symbol:       symbol,
		Base:         strings.ToUpper(base),
		Quote:        strings.ToUpper(quote),
		Margin:       strings.ToUpper(margin),
		ExchangeName: ExchangeName(strings.ToLower(exchangeName)),
		MarketType:   MarketType(strings.ToLower(marketType)),
	}
}

// String 返回币对的字符串表示
func (s *Symbol) String() string {
	if s.Margin != "" {
		return fmt.Sprintf("%s/%s:%s", s.Base, s.Quote, s.Margin)
	}
	return fmt.Sprintf("%s/%s", s.Base, s.Quote)
}

// IsSpot 判断是否为现货市场
func (s *Symbol) IsSpot() bool {
	return s.MarketType == SPOT
}

// IsFutures 判断是否为期货市场
func (s *Symbol) IsFutures() bool {
	return s.MarketType == FUTURESUSDT || s.MarketType == FUTURESCOIN
}

// IsUSDTMargined 判断是否为U本位合约（USDT合约）
func (s *Symbol) IsUSDTMargined() bool {
	return s.MarketType == FUTURESUSDT
}

// IsCoinMargined 判断是否为币本位合约
func (s *Symbol) IsCoinMargined() bool {
	return s.MarketType == FUTURESCOIN
}

// normalizeExchangeName 将字符串转换为ExchangeName类型
func normalizeExchangeName(name string) ExchangeName {
	return ExchangeName(strings.ToLower(strings.TrimSpace(name)))
}

// normalizeMarketType 将字符串转换为MarketType类型
func normalizeMarketType(market string) MarketType {
	market = strings.ToLower(strings.TrimSpace(market))

	// 处理特殊情况的映射
	switch market {
	case "futuresusdt", "futures_usdt", "usdt_futures":
		return FUTURESUSDT
	case "futurescoin", "futures_coin", "coin_futures":
		return FUTURESCOIN
	case "spot":
		return SPOT
	default:
		return MarketType(market)
	}
}

// ParseSymbol 解析币对格式 [a]/[b]:[c] 或 [a]/[b]，自动识别市场类型
// 格式说明：
// - a: 基础币种 (如 BTC)
// - b: 计价币种 (如 USDT)
// - c: 保证金币种 (期货时存在，如 USDT 或 USD)
// 市场类型自动识别规则：
// - 现货: [a]/[b] (如 BTC/USDT)
// - U本位合约: [a]/[b]:[b] (如 BTC/USDT:USDT，quote = margin)
// - 币本位合约: [a]/[b]:[a] (如 BTC/USD:BTC，base = margin)
// 参数说明：
// - symbolStr: 币对字符串，如 "BTC/USDT" 或 "BTC/USDT:USDT"
func ParseSymbol(symbolStr string) (*Symbol, error) {
	symbolStr = strings.TrimSpace(symbolStr)

	// 检查是否包含保证金币种（期货）
	if strings.Contains(symbolStr, ":") {
		// 合约格式：分割为 [base]/[quote] 和 [margin]
		parts := strings.Split(symbolStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid futures symbol format: must be [base]/[quote]:[margin], got: %s", symbolStr)
		}

		baseQuote := parts[0] // [base]/[quote] 部分
		margin := parts[1]    // [margin] 部分

		// 解析基础币种和计价币种
		base, quote, err := parseBaseQuote(baseQuote)
		if err != nil {
			return nil, fmt.Errorf("invalid base/quote format in futures symbol: %w", err)
		}

		// 自动识别市场类型
		var marketType MarketType
		if margin == quote {
			// quote = margin，U本位合约
			marketType = FUTURESUSDT
		} else if margin == base {
			// base = margin，币本位合约
			marketType = FUTURESCOIN
		} else {
			// 其他情况，默认为U本位合约
			marketType = FUTURESUSDT
		}

		// 创建Symbol
		symbol := &Symbol{
			Symbol:     symbolStr,
			Base:       strings.ToUpper(base),
			Quote:      strings.ToUpper(quote),
			Margin:     strings.ToUpper(margin),
			MarketType: marketType,
		}
		return symbol, nil
	}

	// 现货格式：只包含基础币种和计价币种
	base, quote, err := parseBaseQuote(symbolStr)
	if err != nil {
		return nil, fmt.Errorf("invalid spot symbol format: %w", err)
	}

	// 创建Symbol（现货）
	symbol := &Symbol{
		Symbol:     symbolStr,
		Base:       strings.ToUpper(base),
		Quote:      strings.ToUpper(quote),
		Margin:     "",
		MarketType: SPOT,
	}
	return symbol, nil
}

// parseBaseQuote 解析基础币种和计价币种
func parseBaseQuote(baseQuote string) (base, quote string, err error) {
	baseQuote = strings.TrimSpace(baseQuote)

	// 入参一定是 [a]/[b] 格式，如果不是这种格式直接返回错误
	if !strings.Contains(baseQuote, "/") {
		return "", "", fmt.Errorf("invalid format: must be [base]/[quote], got: %s", baseQuote)
	}

	parts := strings.Split(baseQuote, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: must be [base]/[quote], got: %s", baseQuote)
	}

	base = strings.TrimSpace(parts[0])
	quote = strings.TrimSpace(parts[1])

	if base == "" || quote == "" {
		return "", "", fmt.Errorf("invalid format: base and quote cannot be empty, got: %s", baseQuote)
	}

	return base, quote, nil
}

// FormatSymbol 根据Symbol和ExchangeName生成交易所特定的订阅符号
func FormatSymbol(symbol *Symbol, exchangeName ExchangeName) (string, error) {
	if symbol == nil {
		return "", fmt.Errorf("symbol cannot be nil")
	}

	// 根据交易所和市场类型格式化
	var formattedSymbol string

	switch exchangeName {
	case BINANCE:
		formattedSymbol = formatBinanceSymbol(symbol, symbol.MarketType)
	case OKX:
		formattedSymbol = formatOKXSymbol(symbol, string(symbol.MarketType))
	case BYBIT:
		formattedSymbol = formatBybitSymbol(symbol, string(symbol.MarketType))
	case GATE:
		formattedSymbol = formatGateSymbol(symbol, string(symbol.MarketType))
	case MEXC:
		formattedSymbol = formatMEXCSymbol(symbol, string(symbol.MarketType))
	default:
		// 默认格式：base + quote
		formattedSymbol = symbol.Base + symbol.Quote
	}

	return formattedSymbol, nil
}

// FormatSymbolByExchange 根据交易所名称、基础币种、计价币种、保证金币种和市场类型生成交易所特定的订阅符号
func FormatSymbolByExchange(exchangeName ExchangeName, base, quote, margin string, marketType MarketType) (string, error) {
	symbol := &Symbol{
		Base:       strings.ToUpper(base),
		Quote:      strings.ToUpper(quote),
		Margin:     strings.ToUpper(margin),
		MarketType: marketType,
	}
	return FormatSymbol(symbol, exchangeName)
}

// formatBinanceSymbol 格式化Binance币对
func formatBinanceSymbol(symbol *Symbol, marketType MarketType) string {
	switch marketType {
	case SPOT, FUTURESUSDT:
		return symbol.Base + symbol.Quote // BTCUSDT
	case FUTURESCOIN:
		return symbol.Base + symbol.Quote + "_PERP" // BTCUSDT_PERP
	default:
		return symbol.Base + symbol.Quote
	}
}

// formatOKXSymbol 格式化OKX币对
func formatOKXSymbol(symbol *Symbol, marketType string) string {
	switch marketType {
	case "SPOT":
		return symbol.Base + "-" + symbol.Quote // BTC-USDT
	case "FUTURESUSDT":
		return symbol.Base + "-" + symbol.Quote + "-SWAP" // BTC-USDT-SWAP
	case "FUTURESCOIN":
		return symbol.Base + "-" + symbol.Quote + "-SWAP" // BTC-USD-SWAP
	default:
		return symbol.Base + "-" + symbol.Quote
	}
}

// formatBybitSymbol 格式化Bybit币对
func formatBybitSymbol(symbol *Symbol, marketType string) string {
	switch marketType {
	case "SPOT", "FUTURESUSDT", "FUTURESCOIN":
		return symbol.Base + symbol.Quote // BTCUSDT, BTCUSD
	default:
		return symbol.Base + symbol.Quote
	}
}

// formatGateSymbol 格式化Gate币对
func formatGateSymbol(symbol *Symbol, marketType string) string {
	switch marketType {
	case "SPOT", "FUTURESUSDT", "FUTURESCOIN":
		return symbol.Base + "_" + symbol.Quote // BTC_USDT, BTC_USD
	default:
		return symbol.Base + "_" + symbol.Quote
	}
}

// formatMEXCSymbol 格式化MEXC币对
func formatMEXCSymbol(symbol *Symbol, marketType string) string {
	switch marketType {
	case "SPOT", "FUTURESUSDT", "FUTURESCOIN":
		return symbol.Base + symbol.Quote // BTCUSDT, BTCUSD
	default:
		return symbol.Base + symbol.Quote
	}
}

// ReverseParseSymbol 从 base、quote、margin 构建 Symbol 结构
// 参数说明：
// - base: 基础币种 (如 BTC)
// - quote: 计价币种 (如 USDT)
// - margin: 保证金币种 (期货时存在，可能为空)
// - exchangeName: 交易所名称
// - marketType: 市场类型
func ReverseParseSymbol(base, quote, margin, exchangeName, marketType string) (*Symbol, error) {
	if base == "" {
		return nil, fmt.Errorf("base cannot be empty")
	}
	if quote == "" {
		return nil, fmt.Errorf("quote cannot be empty")
	}

	normalizedExchangeName := normalizeExchangeName(exchangeName)
	normalizedMarketType := normalizeMarketType(marketType)

	// 根据市场类型确定保证金币种（如果未提供）
	if margin == "" {
		if normalizedMarketType == FUTURESUSDT {
			margin = quote // U本位合约：保证金币种 = quote币种 (如 BTC/USDT:USDT)
		} else if normalizedMarketType == FUTURESCOIN {
			margin = base // 币本位合约：保证金币种 = base币种 (如 BTC/USD:BTC)
		}
		// 现货市场保持 margin = ""
	}

	// 构建交易所格式的币对符号
	var exchangeSymbol string

	switch normalizedExchangeName {
	case BINANCE:
		exchangeSymbol = formatBinanceSymbol(&Symbol{Base: base, Quote: quote, Margin: margin}, normalizedMarketType)
	case OKX:
		exchangeSymbol = formatOKXSymbol(&Symbol{Base: base, Quote: quote, Margin: margin}, string(normalizedMarketType))
	case BYBIT:
		exchangeSymbol = formatBybitSymbol(&Symbol{Base: base, Quote: quote, Margin: margin}, string(normalizedMarketType))
	case GATE:
		exchangeSymbol = formatGateSymbol(&Symbol{Base: base, Quote: quote, Margin: margin}, string(normalizedMarketType))
	case MEXC:
		exchangeSymbol = formatMEXCSymbol(&Symbol{Base: base, Quote: quote, Margin: margin}, string(normalizedMarketType))
	default:
		// 默认格式：base + quote
		exchangeSymbol = base + quote
	}

	return NewSymbol(exchangeSymbol, base, quote, margin, normalizedExchangeName, normalizedMarketType), nil
}

// reverseParseBinanceSymbol 反解析Binance币对
func reverseParseBinanceSymbol(symbol, marketType string) (base, quote string, err error) {
	symbol = strings.ToUpper(symbol)

	switch strings.ToLower(marketType) {
	case "spot", "futuresusdt", "futures_usdt":
		// BTCUSDT -> BTC, USDT
		if strings.HasSuffix(symbol, "USDT") {
			return strings.TrimSuffix(symbol, "USDT"), "USDT", nil
		}
		// 尝试其他常见计价币种
		quotes := []string{"BTC", "ETH", "BNB", "BUSD", "USDC"}
		for _, q := range quotes {
			if strings.HasSuffix(symbol, q) {
				return strings.TrimSuffix(symbol, q), q, nil
			}
		}
	case "futurescoin", "futures_coin":
		// BTCUSD_PERP -> BTC, USD
		if strings.HasSuffix(symbol, "_PERP") {
			symbol = strings.TrimSuffix(symbol, "_PERP")
			if strings.HasSuffix(symbol, "USD") {
				base := strings.TrimSuffix(symbol, "USD")
				quote := "USD"
				return base, quote, nil
			}
		}
	}

	return smartReverseParse(symbol)
}

// reverseParseOKXSymbol 反解析OKX币对
func reverseParseOKXSymbol(symbol, marketType string) (base, quote string, err error) {
	symbol = strings.ToUpper(symbol)

	// 移除 -SWAP 后缀
	if strings.HasSuffix(symbol, "-SWAP") {
		symbol = strings.TrimSuffix(symbol, "-SWAP")
	}

	// 按破折号分割
	parts := strings.Split(symbol, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("cannot parse OKX symbol: %s", symbol)
}

// reverseParseBybitSymbol 反解析Bybit币对
func reverseParseBybitSymbol(symbol, marketType string) (base, quote string, err error) {
	return smartReverseParse(symbol)
}

// reverseParseGateSymbol 反解析Gate币对
func reverseParseGateSymbol(symbol, marketType string) (base, quote string, err error) {
	symbol = strings.ToUpper(symbol)

	// 按下划线分割
	parts := strings.Split(symbol, "_")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("cannot parse Gate symbol: %s", symbol)
}

// reverseParseMEXCSymbol 反解析MEXC币对
func reverseParseMEXCSymbol(symbol, marketType string) (base, quote string, err error) {
	return smartReverseParse(symbol)
}

// smartReverseParse 智能反解析币对
func smartReverseParse(symbol string) (base, quote string, err error) {
	symbol = strings.ToUpper(symbol)

	// 从常见的计价币种开始匹配
	commonQuotes := []string{"USDT", "BTC", "ETH", "BNB", "BUSD", "USDC", "USD", "EUR", "GBP"}
	for _, q := range commonQuotes {
		if strings.HasSuffix(symbol, q) {
			potentialBase := strings.TrimSuffix(symbol, q)
			if potentialBase != "" && len(potentialBase) >= 2 {
				return potentialBase, q, nil
			}
		}
	}

	return "", "", fmt.Errorf("cannot smart parse symbol: %s", symbol)
}

// ParseExchangeSymbol 从交易所币对符号反解析为 base、quote、margin
func ParseExchangeSymbol(exchangeSymbol, exchangeName, marketType string) (*Symbol, error) {
	if exchangeSymbol == "" {
		return nil, fmt.Errorf("exchange symbol cannot be empty")
	}

	normalizedExchangeName := normalizeExchangeName(exchangeName)
	normalizedMarketType := normalizeMarketType(marketType)

	var base, quote string
	var err error

	// 根据交易所和市场类型反解析
	switch normalizedExchangeName {
	case BINANCE:
		base, quote, err = reverseParseBinanceSymbol(exchangeSymbol, string(normalizedMarketType))
	case OKX:
		base, quote, err = reverseParseOKXSymbol(exchangeSymbol, string(normalizedMarketType))
	case BYBIT:
		base, quote, err = reverseParseBybitSymbol(exchangeSymbol, string(normalizedMarketType))
	case GATE:
		base, quote, err = reverseParseGateSymbol(exchangeSymbol, string(normalizedMarketType))
	case MEXC:
		base, quote, err = reverseParseMEXCSymbol(exchangeSymbol, string(normalizedMarketType))
	default:
		// 默认反解析：尝试智能匹配
		base, quote, err = smartReverseParse(exchangeSymbol)
	}

	if err != nil {
		return nil, err
	}

	// 确定保证金币种
	margin := ""
	if normalizedMarketType == FUTURESUSDT {
		margin = quote // U本位合约：保证金币种 = quote币种 (如 BTC/USDT:USDT)
	} else if normalizedMarketType == FUTURESCOIN {
		margin = base // 币本位合约：保证金币种 = base币种 (如 BTC/USD:BTC)
	}

	return NewSymbol(exchangeSymbol, base, quote, margin, normalizedExchangeName, normalizedMarketType), nil
}

// ConvertSymbol 转换币对格式（从一个交易所格式转换到另一个）
func ConvertSymbol(fromSymbol, fromExchange, fromMarket, toExchange, toMarket string) (string, error) {
	// 先反解析源币对
	symbol, err := ParseExchangeSymbol(fromSymbol, fromExchange, fromMarket)
	if err != nil {
		return "", fmt.Errorf("failed to parse source symbol: %w", err)
	}

	// 再格式化为目标格式
	result, err := FormatSymbol(symbol, ExchangeName(toExchange))
	if err != nil {
		return "", fmt.Errorf("failed to format target symbol: %w", err)
	}

	return result, nil
}
