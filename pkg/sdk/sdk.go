package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/kingsmao/exchange-connector/internal/cache"
	binancefuturescoin "github.com/kingsmao/exchange-connector/internal/exchange/binance/futures_coin"
	binancefuturesusdt "github.com/kingsmao/exchange-connector/internal/exchange/binance/futures_usdt"
	binancespot "github.com/kingsmao/exchange-connector/internal/exchange/binance/spot"
	bybitfuturescoin "github.com/kingsmao/exchange-connector/internal/exchange/bybit/futures_coin"
	bybitfuturesusdt "github.com/kingsmao/exchange-connector/internal/exchange/bybit/futures_usdt"
	bybitspot "github.com/kingsmao/exchange-connector/internal/exchange/bybit/spot"
	gatefuturescoin "github.com/kingsmao/exchange-connector/internal/exchange/gate/futures_coin"
	gatefuturesusdt "github.com/kingsmao/exchange-connector/internal/exchange/gate/futures_usdt"
	gatespot "github.com/kingsmao/exchange-connector/internal/exchange/gate/spot"
	mexcfuturescoin "github.com/kingsmao/exchange-connector/internal/exchange/mexc/futures_coin"
	mexcfuturesusdt "github.com/kingsmao/exchange-connector/internal/exchange/mexc/futures_usdt"
	mexcspot "github.com/kingsmao/exchange-connector/internal/exchange/mexc/spot"
	okxfuturescoin "github.com/kingsmao/exchange-connector/internal/exchange/okx/futures_coin"
	okxfuturesusdt "github.com/kingsmao/exchange-connector/internal/exchange/okx/futures_usdt"
	okxspot "github.com/kingsmao/exchange-connector/internal/exchange/okx/spot"
	"github.com/kingsmao/exchange-connector/internal/manager"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// ExchangeConfig 交易所配置
type ExchangeConfig struct {
	Name   schema.ExchangeName // 交易所名称
	Market schema.MarketType   // 市场类型
	Weight int                 // 权重
}

// SymbolConfig 币对配置
type SymbolConfig struct {
	Base   string            // 基础货币
	Quote  string            // 计价货币
	Margin string            // 保证金币种（期货时存在）
	Market schema.MarketType // 市场类型
}

// SDK provides a high-level interface for exchange operations
type SDK struct {
	manager *manager.Manager
	// 配置存储
	exchangeConfigs []ExchangeConfig
	symbolConfigs   []SymbolConfig
}

// NewSDK creates a new SDK instance
func NewSDK() *SDK {
	return &SDK{
		manager: manager.NewManager(),
	}
}

// GetExchangeConfigs returns all exchange configurations
func (sdk *SDK) GetExchangeConfigs() []ExchangeConfig {
	return sdk.exchangeConfigs
}

// GetExchangeConfig returns specific exchange configuration
func (sdk *SDK) GetExchangeConfig(name schema.ExchangeName, market schema.MarketType) (*ExchangeConfig, bool) {
	index, config := sdk.findExchangeConfig(name, market)
	if index == -1 {
		return nil, false
	}
	return config, true
}

// RemoveExchange removes an exchange by setting its weight to 0
func (sdk *SDK) RemoveExchange(name schema.ExchangeName, market schema.MarketType) error {
	return sdk.AddExchange(ExchangeConfig{
		Name:   name,
		Market: market,
		Weight: 0,
	})
}

// GetActiveExchanges returns all currently active exchanges from manager
func (sdk *SDK) GetActiveExchanges() []string {
	var activeExchanges []string
	configs := sdk.GetExchangeConfigs()
	for _, config := range configs {
		activeExchanges = append(activeExchanges, string(config.Name)+":"+string(config.Market))
	}
	return activeExchanges
}

// IsExchangeActive checks if an exchange is currently active
func (sdk *SDK) IsExchangeActive(name schema.ExchangeName, market schema.MarketType) bool {
	_, exists := sdk.GetExchangeConfig(name, market)
	return exists
}

// findExchangeConfig 查找现有的交易所配置
func (sdk *SDK) findExchangeConfig(name schema.ExchangeName, market schema.MarketType) (int, *ExchangeConfig) {
	for i, config := range sdk.exchangeConfigs {
		if config.Name == name && config.Market == market {
			return i, &config
		}
	}
	return -1, nil
}

// AddExchange 添加或更新交易所配置
func (sdk *SDK) AddExchange(config ExchangeConfig) error {
	// 1. 检查权重值
	if config.Weight == 0 {
		// 权重为0，尝试删除现有交易所
		if index, _ := sdk.findExchangeConfig(config.Name, config.Market); index != -1 {
			// 从配置列表中删除
			sdk.exchangeConfigs = append(sdk.exchangeConfigs[:index], sdk.exchangeConfigs[index+1:]...)
			// 从manager中删除交易所并断开WebSocket连接
			if err := sdk.manager.RemoveExchange(config.Name, config.Market); err != nil {
				logger.Warn("删除交易所 %s %s 失败: %v", config.Name, config.Market, err)
			}
			return nil
		}
		// 如果不存在，直接返回（无需操作）
		return nil
	}

	// 2. 检查是否已存在
	if index, existingConfig := sdk.findExchangeConfig(config.Name, config.Market); index != -1 {
		// 已存在，检查权重是否相同
		if existingConfig.Weight == config.Weight {
			// 权重相同，无需操作
			return nil
		}

		// 权重不同，更新权重
		// 更新配置
		sdk.exchangeConfigs[index].Weight = config.Weight
		// 更新manager中的权重
		if err := sdk.manager.UpdateExchangeWeight(config.Name, config.Market, config.Weight); err != nil {
			logger.Warn("更新交易所 %s %s 权重失败: %v", config.Name, config.Market, err)
		}
		return nil
	}

	// 3. 不存在，创建新的交易所实例
	exchange := sdk.createExchange(config)
	if exchange == nil {
		return fmt.Errorf("failed to create exchange instance for %s %s", config.Name, config.Market)
	}

	// 4. 添加到manager
	sdk.manager.AddExchange(exchange, config.Weight)

	// 5. 保存配置
	sdk.exchangeConfigs = append(sdk.exchangeConfigs, config)

	return nil
}

// AddExchanges 批量添加交易所配置
func (sdk *SDK) AddExchanges(configs []ExchangeConfig) error {
	for _, config := range configs {
		if err := sdk.AddExchange(config); err != nil {
			return fmt.Errorf("failed to add exchange %s %s: %w", config.Name, config.Market, err)
		}
	}
	return nil
}

// AddSymbols 批量添加币对配置
func (sdk *SDK) AddSymbols(configs []SymbolConfig) {
	sdk.symbolConfigs = append(sdk.symbolConfigs, configs...)
}

// AddSymbolsByExchange 按交易所批量添加币对（自动识别市场类型）
func (sdk *SDK) AddSymbolsByExchange(exchange schema.ExchangeName, symbols []string) {
	for _, symbolStr := range symbols {
		// 使用现有的ParseSymbol函数解析币对格式 [base]/[quote]:[margin]
		parsedSymbol, err := schema.ParseSymbol(symbolStr)
		if err != nil {
			// 如果解析失败，记录错误但继续处理其他币对
			continue
		}

		sdk.symbolConfigs = append(sdk.symbolConfigs, SymbolConfig{
			Base:   parsedSymbol.Base,
			Quote:  parsedSymbol.Quote,
			Margin: parsedSymbol.Margin,
			Market: parsedSymbol.MarketType,
		})
	}
}

// AddSymbolsAndSubscribe 添加币对并自动订阅WebSocket（一步完成）
func (sdk *SDK) AddSymbolsAndSubscribe(ctx context.Context, symbols []string) error {
	// 1. 添加币对
	for _, symbolStr := range symbols {
		// 使用现有的ParseSymbol函数解析币对格式 [base]/[quote]:[margin]
		// 自动识别市场类型：现货为 [base]/[quote]，合约为 [base]/[quote]:[margin]
		parsedSymbol, err := schema.ParseSymbol(symbolStr)
		if err != nil {
			// 如果解析失败，记录错误但继续处理其他币对
			continue
		}

		sdk.symbolConfigs = append(sdk.symbolConfigs, SymbolConfig{
			Base:   parsedSymbol.Base,
			Quote:  parsedSymbol.Quote,
			Margin: parsedSymbol.Margin,
			Market: parsedSymbol.MarketType,
		})
	}

	// 2. 自动订阅
	return sdk.autoSubscribe(ctx)
}

// AddSymbolsByExchangeAndSubscribe 按交易所添加币对并自动订阅WebSocket（一步完成）
func (sdk *SDK) AddSymbolsByExchangeAndSubscribe(ctx context.Context, exchange schema.ExchangeName, symbols []string) error {
	// 1. 按交易所添加币对
	sdk.AddSymbolsByExchange(exchange, symbols)

	// 2. 自动订阅
	return sdk.autoSubscribe(ctx)
}

// autoSubscribe 自动订阅所有配置的交易所和币对（内部方法）
func (sdk *SDK) autoSubscribe(ctx context.Context) error {
	// 1. 启动所有WebSocket连接
	if err := sdk.manager.StartWS(ctx); err != nil {
		return err
	}

	// 3. 自动订阅所有币对（按市场类型分组，批量订阅）
	// 按交易所和市场类型分组币对
	subscriptionGroups := make(map[string][]string) // key: "exchangeName_marketType"

	for _, symbolConfig := range sdk.symbolConfigs {
		// 找到支持该市场的交易所
		for _, exchangeConfig := range sdk.exchangeConfigs {
			if exchangeConfig.Market == symbolConfig.Market {
				// 生成分组key（使用分隔符避免市场类型中的下划线问题）
				groupKey := fmt.Sprintf("%s|%s", exchangeConfig.Name, exchangeConfig.Market)

				// 使用FormatSymbolByExchange格式化币对名称
				formattedSymbol, err := schema.FormatSymbolByExchange(
					exchangeConfig.Name,
					symbolConfig.Base,
					symbolConfig.Quote,
					symbolConfig.Margin,
					exchangeConfig.Market,
				)
				if err != nil {
					logger.Warn("格式化币对符号失败 %s/%s: %v", symbolConfig.Base, symbolConfig.Quote, err)
					continue
				}

				// 添加到对应的分组
				subscriptionGroups[groupKey] = append(subscriptionGroups[groupKey], formattedSymbol)
			}
		}
	}

	// 批量订阅每个分组
	for groupKey, symbols := range subscriptionGroups {
		// 解析分组key
		parts := strings.Split(groupKey, "|")
		if len(parts) != 2 {
			logger.Warn("无效的分组key: %s", groupKey)
			continue
		}

		exchangeName := schema.ExchangeName(parts[0])
		marketType := schema.MarketType(parts[1])

		if len(symbols) == 0 {
			continue
		}

		logger.Info("批量订阅 %s %s: %v", exchangeName, marketType, symbols)

		// 批量订阅K线数据（固定1m）
		if err := sdk.manager.SubscribeKline(ctx, exchangeName, marketType, symbols); err != nil {
			logger.Warn("订阅K线数据失败 %s %s: %v", exchangeName, marketType, err)
		}

		// 批量订阅深度数据
		if err := sdk.manager.SubscribeDepth(ctx, exchangeName, marketType, symbols); err != nil {
			logger.Warn("订阅深度数据失败 %s %s: %v", exchangeName, marketType, err)
		}
	}

	return nil
}

// createExchange 根据配置创建交易所实例
func (sdk *SDK) createExchange(config ExchangeConfig) interfaces.Exchange {
	// 获取缓存实例
	cache := sdk.manager.Cache()

	switch config.Name {
	case schema.BINANCE:
		return sdk.createBinanceExchange(config.Market, cache)
	case schema.OKX:
		return sdk.createOKXExchange(config.Market, cache)
	case schema.BYBIT:
		return sdk.createBybitExchange(config.Market, cache)
	case schema.GATE:
		return sdk.createGateExchange(config.Market, cache)
	case schema.MEXC:
		return sdk.createMEXCExchange(config.Market, cache)
	default:
		logger.Warn("不支持的交易所: %s", config.Name)
		return nil
	}
}

// createBinanceExchange 创建Binance交易所实例
func (sdk *SDK) createBinanceExchange(market schema.MarketType, cache *cache.MemoryCache) interfaces.Exchange {
	switch market {
	case schema.SPOT:
		return binancespot.NewSpotExchange(cache)
	case schema.FUTURESUSDT:
		return binancefuturesusdt.NewFuturesUSDTExchange(cache)
	case schema.FUTURESCOIN:
		return binancefuturescoin.NewFuturesCoinExchange(cache)
	default:
		logger.Warn("Binance不支持的市场类型: %s", market)
		return nil
	}
}

// createOKXExchange 创建OKX交易所实例
func (sdk *SDK) createOKXExchange(market schema.MarketType, cache *cache.MemoryCache) interfaces.Exchange {
	switch market {
	case schema.SPOT:
		return okxspot.NewSpotExchange(cache)
	case schema.FUTURESUSDT:
		return okxfuturesusdt.NewFuturesUSDTExchange(cache)
	case schema.FUTURESCOIN:
		return okxfuturescoin.NewFuturesCoinExchange(cache)
	default:
		logger.Warn("OKX不支持的市场类型: %s", market)
		return nil
	}
}

// createBybitExchange 创建Bybit交易所实例
func (sdk *SDK) createBybitExchange(market schema.MarketType, cache *cache.MemoryCache) interfaces.Exchange {
	switch market {
	case schema.SPOT:
		return bybitspot.NewSpotExchange(cache)
	case schema.FUTURESUSDT:
		return bybitfuturesusdt.NewFuturesUSDTExchange(cache)
	case schema.FUTURESCOIN:
		return bybitfuturescoin.NewFuturesCoinExchange(cache)
	default:
		logger.Warn("Bybit不支持的市场类型: %s", market)
		return nil
	}
}

// createGateExchange 创建Gate交易所实例
func (sdk *SDK) createGateExchange(market schema.MarketType, cache *cache.MemoryCache) interfaces.Exchange {
	switch market {
	case schema.SPOT:
		return gatespot.NewSpotExchange(cache)
	case schema.FUTURESUSDT:
		return gatefuturesusdt.NewFuturesUSDTExchange(cache)
	case schema.FUTURESCOIN:
		return gatefuturescoin.NewFuturesCoinExchange(cache)
	default:
		logger.Warn("Gate不支持的市场类型: %s", market)
		return nil
	}
}

// createMEXCExchange 创建MEXC交易所实例
func (sdk *SDK) createMEXCExchange(market schema.MarketType, cache *cache.MemoryCache) interfaces.Exchange {
	switch market {
	case schema.SPOT:
		return mexcspot.NewSpotExchange(cache)
	case schema.FUTURESUSDT:
		return mexcfuturesusdt.NewFuturesUSDTExchange(cache)
	case schema.FUTURESCOIN:
		return mexcfuturescoin.NewFuturesCoinExchange(cache)
	default:
		logger.Warn("MEXC不支持的市场类型: %s", market)
		return nil
	}
}

// SubscribeKline subscribes to kline data for specified symbols (fixed to 1m interval)
func (sdk *SDK) SubscribeKline(ctx context.Context, name schema.ExchangeName, market schema.MarketType, symbols []string) error {
	return sdk.manager.SubscribeKline(ctx, name, market, symbols)
}

// SubscribeDepth subscribes to depth data for specified symbols
func (sdk *SDK) SubscribeDepth(ctx context.Context, name schema.ExchangeName, market schema.MarketType, symbols []string) error {
	return sdk.manager.SubscribeDepth(ctx, name, market, symbols)
}

// FetchDepth fetches depth data from REST API
func (sdk *SDK) FetchDepth(ctx context.Context, market schema.MarketType, base, quote string, limit int) (schema.Depth, error) {
	return sdk.manager.FetchDepth(ctx, market, base, quote, limit)
}

// WatchKline 根据币对符号智能读取K线数据（自动判断市场类型，按默认顺序查找）
func (sdk *SDK) WatchKline(symbol string) (schema.Kline, bool) {
	// 解析币对符号
	parsedSymbol, err := schema.ParseSymbol(symbol)
	if err != nil {
		logger.Warn("解析币对符号失败 %s: %v", symbol, err)
		return schema.Kline{}, false
	}

	// 按默认顺序查找数据
	exchanges := sdk.getDefaultExchangeOrder()
	for _, exchange := range exchanges {
		formattedSymbol, err := schema.FormatSymbolByExchange(
			exchange,
			parsedSymbol.Base,
			parsedSymbol.Quote,
			parsedSymbol.Margin,
			parsedSymbol.MarketType,
		)
		if err != nil {
			return schema.Kline{}, false
		}
		if kline, ok := sdk.manager.WatchKline(exchange, parsedSymbol.MarketType, formattedSymbol); ok {
			return kline, true
		}
	}

	return schema.Kline{}, false
}

// WatchDepth 根据币对符号智能读取深度数据（自动判断市场类型，按默认顺序查找）
func (sdk *SDK) WatchDepth(symbol string) (schema.Depth, bool) {
	// 解析币对符号
	parsedSymbol, err := schema.ParseSymbol(symbol)
	if err != nil {
		logger.Warn("解析币对符号失败 %s: %v", symbol, err)
		return schema.Depth{}, false
	}

	// 按默认顺序查找数据
	exchanges := sdk.getDefaultExchangeOrder()
	for _, exchange := range exchanges {
		formattedSymbol, err := schema.FormatSymbolByExchange(
			exchange,
			parsedSymbol.Base,
			parsedSymbol.Quote,
			parsedSymbol.Margin,
			parsedSymbol.MarketType,
		)
		if err != nil {
			return schema.Depth{}, false
		}
		if depth, ok := sdk.manager.WatchDepth(exchange, parsedSymbol.MarketType, formattedSymbol); ok {
			return depth, true
		}
	}

	return schema.Depth{}, false
}

// getDefaultExchangeOrder 获取默认的交易所查找顺序
func (sdk *SDK) getDefaultExchangeOrder() []schema.ExchangeName {
	return []schema.ExchangeName{
		schema.BINANCE, // 第一位：Binance
		schema.OKX,     // 第二位：OKX
		schema.BYBIT,   // 第三位：Bybit
		schema.GATE,    // 第四位：Gate
		schema.MEXC,    // 第五位：MEXC
	}
}

// StartWS starts all exchange WebSocket connections
func (sdk *SDK) StartWS(ctx context.Context) error {
	return sdk.manager.StartWS(ctx)
}
