package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// ExchangeInfoCache 管理交易规则信息的内存缓存
type ExchangeInfoCache struct {
	mu    sync.RWMutex
	cache map[string]schema.ExchangeInfo // key: "exchangeName:marketType"
}

// NewExchangeInfoCache 创建新的交易规则信息缓存
func NewExchangeInfoCache() *ExchangeInfoCache {
	return &ExchangeInfoCache{
		cache: make(map[string]schema.ExchangeInfo),
	}
}

// GetCacheKey 生成缓存键
func (c *ExchangeInfoCache) GetCacheKey(exchangeName schema.ExchangeName, marketType schema.MarketType) string {
	return fmt.Sprintf("%s:%s", exchangeName, marketType)
}

// Set 设置交易规则信息到缓存
func (c *ExchangeInfoCache) Set(exchangeInfo schema.ExchangeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.GetCacheKey(exchangeInfo.Exchange, exchangeInfo.Market)
	exchangeInfo.UpdatedAt = time.Now()
	c.cache[key] = exchangeInfo

	logger.Info("交易规则信息已缓存: %s, 交易对数量: %d", key, len(exchangeInfo.Symbols))
}

// Get 从缓存获取交易规则信息
func (c *ExchangeInfoCache) Get(exchangeName schema.ExchangeName, marketType schema.MarketType) (schema.ExchangeInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.GetCacheKey(exchangeName, marketType)
	info, exists := c.cache[key]
	return info, exists
}

// GetSymbol 从缓存获取特定交易对信息
func (c *ExchangeInfoCache) GetSymbol(exchangeName schema.ExchangeName, marketType schema.MarketType, symbol string) (*schema.Symbol, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.GetCacheKey(exchangeName, marketType)
	info, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	for i := range info.Symbols {
		if info.Symbols[i].Symbol == symbol {
			return &info.Symbols[i], true
		}
	}

	return nil, false
}

// GetAllSymbols 获取指定交易所和市场的所有交易对
func (c *ExchangeInfoCache) GetAllSymbols(exchangeName schema.ExchangeName, marketType schema.MarketType) ([]schema.Symbol, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.GetCacheKey(exchangeName, marketType)
	info, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	return info.Symbols, true
}

// IsExpired 检查缓存是否过期（默认24小时）
func (c *ExchangeInfoCache) IsExpired(exchangeName schema.ExchangeName, marketType schema.MarketType, expireDuration time.Duration) bool {
	if expireDuration == 0 {
		expireDuration = 24 * time.Hour // 默认24小时过期
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.GetCacheKey(exchangeName, marketType)
	info, exists := c.cache[key]
	if !exists {
		return true // 不存在视为过期
	}

	return time.Since(info.UpdatedAt) > expireDuration
}

// Refresh 刷新交易规则信息（通过REST API）
func (c *ExchangeInfoCache) Refresh(ctx context.Context, exchangeName schema.ExchangeName, marketType schema.MarketType, restClient interface{}) error {
	// 检查 REST 客户端是否实现了 GetExchangeInfo 方法
	type ExchangeInfoGetter interface {
		GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error)
	}

	client, ok := restClient.(ExchangeInfoGetter)
	if !ok {
		return fmt.Errorf("REST client does not support GetExchangeInfo method")
	}

	logger.Info("正在刷新交易规则信息: %s:%s", exchangeName, marketType)

	exchangeInfo, err := client.GetExchangeInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch exchange info: %w", err)
	}

	// 设置交易所和市场类型信息
	exchangeInfo.Exchange = exchangeName
	exchangeInfo.Market = marketType

	c.Set(exchangeInfo)
	return nil
}

// RefreshIfExpired 如果缓存过期则刷新
func (c *ExchangeInfoCache) RefreshIfExpired(ctx context.Context, exchangeName schema.ExchangeName, marketType schema.MarketType, restClient interface{}, expireDuration time.Duration) error {
	if c.IsExpired(exchangeName, marketType, expireDuration) {
		return c.Refresh(ctx, exchangeName, marketType, restClient)
	}
	return nil
}

// Clear 清空指定的缓存
func (c *ExchangeInfoCache) Clear(exchangeName schema.ExchangeName, marketType schema.MarketType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.GetCacheKey(exchangeName, marketType)
	delete(c.cache, key)

	logger.Info("已清空交易规则信息缓存: %s", key)
}

// ClearAll 清空所有缓存
func (c *ExchangeInfoCache) ClearAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]schema.ExchangeInfo)
	logger.Info("已清空所有交易规则信息缓存")
}

// GetCacheStats 获取缓存统计信息
func (c *ExchangeInfoCache) GetCacheStats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]int)
	for key, info := range c.cache {
		stats[key] = len(info.Symbols)
	}

	return stats
}
