package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// ExchangeInfo holds exchange information including weight
type ExchangeInfo struct {
	Exchange interfaces.Exchange
	Weight   int
}

// Manager coordinates exchanges and exposes read APIs backed by cache.
type Manager struct {
	cache     *cache.MemoryCache
	exchanges map[string]*ExchangeInfo
	mu        sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		cache:     cache.NewMemoryCache(),
		exchanges: make(map[string]*ExchangeInfo),
	}
}

func (m *Manager) AddExchange(ex interfaces.Exchange, weight int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := string(ex.Name()) + ":" + string(ex.Market())
	m.exchanges[key] = &ExchangeInfo{
		Exchange: ex,
		Weight:   weight,
	}
}

// RemoveExchange removes an exchange and stops its WebSocket connection
func (m *Manager) RemoveExchange(name schema.ExchangeName, market schema.MarketType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(name) + ":" + string(market)
	exInfo, exists := m.exchanges[key]
	if !exists {
		return fmt.Errorf("exchange %s %s not found", name, market)
	}

	// 停止WebSocket连接
	if exInfo.Exchange.WS() != nil {
		// 断开WebSocket连接
		if err := exInfo.Exchange.WS().Close(); err != nil {
			logger.Warn("交易所 %s %s WebSocket 断开失败: %v", name, market, err)
		} else {
			logger.Info("交易所 %s %s WebSocket 已断开", name, market)
		}
	}

	// 从map中删除
	delete(m.exchanges, key)
	logger.Info("交易所 %s %s 已从manager中删除", name, market)

	return nil
}

// UpdateExchangeWeight updates the weight of an existing exchange
func (m *Manager) UpdateExchangeWeight(name schema.ExchangeName, market schema.MarketType, newWeight int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(name) + ":" + string(market)
	exInfo, exists := m.exchanges[key]
	if !exists {
		return fmt.Errorf("exchange %s %s not found", name, market)
	}

	exInfo.Weight = newWeight
	logger.Info("交易所 %s %s 权重已更新为 %d", name, market, newWeight)

	return nil
}

func (m *Manager) GetExchange(name schema.ExchangeName, market schema.MarketType) (interfaces.Exchange, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exInfo, ok := m.exchanges[string(name)+":"+string(market)]
	if !ok {
		return nil, false
	}
	return exInfo.Exchange, true
}

func (m *Manager) GetExchangeInfo(name schema.ExchangeName, market schema.MarketType) (*ExchangeInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exInfo, ok := m.exchanges[string(name)+":"+string(market)]
	return exInfo, ok
}

func (m *Manager) Cache() *cache.MemoryCache { return m.cache }

// StartWS starts all exchange WS loops.
func (m *Manager) StartWS(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedExchanges []string
	var successCount int

	for _, exInfo := range m.exchanges {
		if exInfo.Exchange.WS() == nil {
			continue
		}

		wg.Add(1)
		go func(e *ExchangeInfo) {
			defer wg.Done()

			exchangeName := string(e.Exchange.Name())
			marketType := string(e.Exchange.Market())

			// 尝试连接
			if err := e.Exchange.WS().Connect(ctx); err != nil {
				mu.Lock()
				failedExchanges = append(failedExchanges, fmt.Sprintf("%s-%s", exchangeName, marketType))
				mu.Unlock()
				logger.Error("交易所 %s-%s WebSocket 连接失败: %v", exchangeName, marketType, err)
				return
			}

			// 成功启动
			mu.Lock()
			successCount++
			mu.Unlock()
			logger.Info("交易所 %s-%s WebSocket 连接成功", exchangeName, marketType)

			// 启动消息读取（后台运行，不等待完成）
			go func() {
				if err := e.Exchange.WS().StartReading(ctx); err != nil {
					logger.Error("交易所 %s-%s WebSocket 启动失败: %v", exchangeName, marketType, err)
				}
			}()

			// 启动健康检查（后台运行，不等待完成）
			go func() {
				if err := e.Exchange.WS().StartHealthCheck(ctx); err != nil {
					logger.Warn("交易所 %s-%s WebSocket 健康检查启动失败: %v", exchangeName, marketType, err)
				}
			}()
		}(exInfo)
	}

	wg.Wait()

	// 输出启动结果统计
	if len(failedExchanges) > 0 {
		logger.Info("WebSocket 启动完成: %d 个成功, %d 个失败", successCount, len(failedExchanges))
		logger.Warn("失败的交易所: %v", failedExchanges)
	} else {
		logger.Info("所有交易所 WebSocket 启动成功: %d 个", successCount)
	}

	// 只要有成功的交易所就返回成功
	if successCount > 0 {
		return nil
	}

	// 所有交易所都失败时才返回错误
	return fmt.Errorf("所有交易所 WebSocket 启动失败")
}

// Subscribe simple façade methods (kline/depth)

func (m *Manager) SubscribeKline(ctx context.Context, name schema.ExchangeName, market schema.MarketType, symbols []string) error {
	ex, ok := m.GetExchange(name, market)
	if !ok || ex.WS() == nil {
		return errors.New("ws exchange not found")
	}

	// 固定订阅1m K线数据
	return ex.WS().SubscribeKline(ctx, symbols)
}

func (m *Manager) SubscribeDepth(ctx context.Context, name schema.ExchangeName, market schema.MarketType, symbols []string) error {
	ex, ok := m.GetExchange(name, market)
	if !ok || ex.WS() == nil {
		return errors.New("ws exchange not found")
	}

	// Symbols are already formatted, use directly
	return ex.WS().SubscribeDepth(ctx, symbols)
}

// FetchDepth fetches depth data from REST API
func (m *Manager) FetchDepth(ctx context.Context, market schema.MarketType, base, quote string, limit int) (schema.Depth, error) {
	// Try to get from any available exchange for this market
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, exInfo := range m.exchanges {
		if exInfo.Exchange.Market() == market && exInfo.Exchange.REST() != nil {
			symbol := m.formatSymbol(exInfo.Exchange.Name(), market, base, quote)
			depth, err := exInfo.Exchange.REST().GetDepth(ctx, symbol, limit)
			if err == nil {
				// Cache the result
				m.cache.SetDepth(depth)
				return depth, nil
			}
		}
	}

	return schema.Depth{}, errors.New("no exchange available for fetching depth")
}

// WatchKline returns kline data from WebSocket subscriptions
func (m *Manager) WatchKline(exchange schema.ExchangeName, market schema.MarketType, symbol string) (schema.Kline, bool) {
	if klines, ok := m.cache.GetKline(exchange, market, symbol, "1m"); ok && len(klines) > 0 {
		return klines[0], true // 返回最新的一条K线数据
	}
	// 如果没有找到数据，返回空K线
	return schema.Kline{}, false
}

// WatchDepth returns depth data from WebSocket subscriptions
func (m *Manager) WatchDepth(exchange schema.ExchangeName, market schema.MarketType, symbol string) (schema.Depth, bool) {
	if depth, ok := m.cache.GetDepth(exchange, market, symbol); ok {
		return depth, true
	}
	// 如果没有找到数据，返回空深度
	return schema.Depth{}, false
}

// formatSymbol formats base and quote into exchange-specific symbol format
func (m *Manager) formatSymbol(name schema.ExchangeName, market schema.MarketType, base, quote string) string {
	switch name {
	case schema.BINANCE:
		switch market {
		case schema.SPOT, schema.FUTURESUSDT:
			return base + quote // BTCUSDT
		case schema.FUTURESCOIN:
			return base + quote + "_PERP" // BTCUSD_PERP
		}
	case schema.OKX:
		switch market {
		case schema.SPOT:
			return base + "-" + quote // BTC-USDT
		case schema.FUTURESUSDT:
			return base + "-" + quote + "-SWAP" // BTC-USDT-SWAP
		case schema.FUTURESCOIN:
			return base + "-" + quote + "-SWAP" // BTC-USD-SWAP
		}
	case schema.BYBIT:
		switch market {
		case schema.SPOT, schema.FUTURESUSDT:
			return base + quote // BTCUSDT
		case schema.FUTURESCOIN:
			return base + quote // BTCUSD
		}
	case schema.GATE:
		switch market {
		case schema.SPOT, schema.FUTURESUSDT:
			return base + "_" + quote // BTC_USDT
		case schema.FUTURESCOIN:
			return base + "_" + quote // BTC_USD
		}
	case schema.MEXC:
		switch market {
		case schema.SPOT, schema.FUTURESUSDT:
			return base + quote // BTCUSDT
		case schema.FUTURESCOIN:
			return base + quote // BTCUSD
		}
	}

	// Default fallback
	return base + quote
}
