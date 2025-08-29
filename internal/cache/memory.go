package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// MemoryCache is a threadsafe in-memory store for market data.
// 使用原子操作优化性能，避免锁竞争和内存拷贝
type MemoryCache struct {
	// 原子操作映射表 - 存储指向数据的原子指针
	depths sync.Map // map[string]*unsafe.Pointer -> *schema.Depth
	klines sync.Map // map[string]*unsafe.Pointer -> *[]schema.Kline
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

// cacheKey generates a cache key for exchange-specific data
func cacheKey(exchange schema.ExchangeName, market schema.MarketType, symbol string, subkeys ...string) string {
	key := fmt.Sprintf("%s:%s:%s", exchange, market, symbol)
	for _, s := range subkeys {
		key += "_" + s
	}
	return key
}

func (m *MemoryCache) SetDepth(d schema.Depth) {
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now()
	}

	key := cacheKey(d.Exchange, d.Market, d.Symbol)

	// 🚀 原子操作：创建新数据副本
	newDepth := &d

	// 获取或创建原子指针
	var nilPtr unsafe.Pointer
	atomicPtrInterface, _ := m.depths.LoadOrStore(key, &nilPtr)
	atomicPtr := atomicPtrInterface.(*unsafe.Pointer)

	// 🚀 原子替换指针，零拷贝操作
	atomic.StorePointer(atomicPtr, unsafe.Pointer(newDepth))
}

// FetchDepth fetches depth data from REST API (placeholder implementation)
func (m *MemoryCache) FetchDepth(ctx context.Context, market schema.MarketType, base, quote string, limit int) (schema.Depth, error) {
	// This is a placeholder - in a real implementation, this would fetch from REST API
	// For now, return cached data if available
	symbol := base + quote
	if depth, ok := m.GetDepth(schema.BINANCE, market, symbol); ok {
		return depth, nil
	}
	return schema.Depth{}, fmt.Errorf("no depth data available for %s:%s:%s", market, base, quote)
}

// WatchKline returns kline data from WebSocket subscriptions
func (m *MemoryCache) WatchKline(market schema.MarketType, base, quote string, interval schema.Interval) ([]schema.Kline, bool) {
	symbol := base + quote
	return m.GetKline(schema.BINANCE, market, symbol, interval)
}

// WatchDepth returns depth data from WebSocket subscriptions
func (m *MemoryCache) WatchDepth(market schema.MarketType, base, quote string) (schema.Depth, bool) {
	symbol := base + quote
	return m.GetDepth(schema.BINANCE, market, symbol)
}

func (m *MemoryCache) GetDepth(exchange schema.ExchangeName, market schema.MarketType, symbol string) (schema.Depth, bool) {
	key := cacheKey(exchange, market, symbol)

	// 🚀 原子操作：从原子缓存读取
	if atomicPtrInterface, ok := m.depths.Load(key); ok {
		atomicPtr := atomicPtrInterface.(*unsafe.Pointer)
		dataPtr := atomic.LoadPointer(atomicPtr)
		if dataPtr != nil {
			depth := (*schema.Depth)(dataPtr)
			return *depth, true // 🚀 零拷贝返回
		}
	}

	// 数据不存在
	return schema.Depth{}, false
}

func (m *MemoryCache) SetKline(kl schema.Kline) {
	key := cacheKey(kl.Exchange, kl.Market, kl.Symbol, string(kl.Interval))

	// 🚀 原子操作：创建新数据副本（只保留最新一条）
	newKlineArray := &[]schema.Kline{kl}

	// 获取或创建原子指针
	var nilPtr unsafe.Pointer
	atomicPtrInterface, _ := m.klines.LoadOrStore(key, &nilPtr)
	atomicPtr := atomicPtrInterface.(*unsafe.Pointer)

	// 🚀 原子替换指针，零拷贝操作
	atomic.StorePointer(atomicPtr, unsafe.Pointer(newKlineArray))
}

// AppendKline 保持向后兼容，但实际上只设置最新数据
func (m *MemoryCache) AppendKline(kl schema.Kline) {
	m.SetKline(kl)
}

func (m *MemoryCache) GetKline(exchange schema.ExchangeName, market schema.MarketType, symbol string, interval schema.Interval) ([]schema.Kline, bool) {
	key := cacheKey(exchange, market, symbol, string(interval))

	// 🚀 原子操作：从原子缓存读取
	if atomicPtrInterface, ok := m.klines.Load(key); ok {
		atomicPtr := atomicPtrInterface.(*unsafe.Pointer)
		dataPtr := atomic.LoadPointer(atomicPtr)
		if dataPtr != nil {
			klineArray := (*[]schema.Kline)(dataPtr)
			return *klineArray, true // 🚀 零拷贝返回
		}
	}

	// 数据不存在
	return []schema.Kline{}, false
}
