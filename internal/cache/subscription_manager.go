package cache

import (
	"sync"

	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// SubscriptionManagerImpl implements SubscriptionManager interface
type SubscriptionManagerImpl struct {
	mu sync.RWMutex

	// subscribed symbols for kline
	klineSymbols map[string]struct{}
	// subscribed symbols for depth
	depthSymbols map[string]struct{}
	// kline interval for all symbols (all symbols use the same interval)
	klineInterval schema.Interval
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() interfaces.SubscriptionManager {
	return &SubscriptionManagerImpl{
		klineSymbols:  make(map[string]struct{}),
		depthSymbols:  make(map[string]struct{}),
		klineInterval: schema.Interval1m, // default interval
	}
}

// SubscribeSymbols adds symbols to all subscriptions (kline, depth, etc.), returns newly added symbols
func (sm *SubscriptionManagerImpl) SubscribeSymbols(symbols []string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var newlyAdded []string
	for _, symbol := range symbols {
		if _, exists := sm.klineSymbols[symbol]; !exists {
			sm.klineSymbols[symbol] = struct{}{}
			newlyAdded = append(newlyAdded, symbol)
		}
		if _, exists := sm.depthSymbols[symbol]; !exists {
			sm.depthSymbols[symbol] = struct{}{}
			newlyAdded = append(newlyAdded, symbol)
		}
	}

	return newlyAdded
}

// UnsubscribeSymbols removes symbols from all subscriptions, returns actually removed symbols
func (sm *SubscriptionManagerImpl) UnsubscribeSymbols(symbols []string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var actuallyRemoved []string
	for _, symbol := range symbols {
		if _, exists := sm.klineSymbols[symbol]; exists {
			delete(sm.klineSymbols, symbol)
			actuallyRemoved = append(actuallyRemoved, symbol)
		}
		if _, exists := sm.depthSymbols[symbol]; exists {
			delete(sm.depthSymbols, symbol)
			actuallyRemoved = append(actuallyRemoved, symbol)
		}
	}
	return actuallyRemoved
}

// GetSubscribedSymbols returns all currently subscribed symbols
func (sm *SubscriptionManagerImpl) GetSubscribedSymbols() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbols := make([]string, 0, len(sm.klineSymbols)+len(sm.depthSymbols))
	for symbol := range sm.klineSymbols {
		symbols = append(symbols, symbol)
	}
	for symbol := range sm.depthSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// ClearAll clears all subscriptions
func (sm *SubscriptionManagerImpl) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.klineSymbols = make(map[string]struct{})
	sm.depthSymbols = make(map[string]struct{})
	sm.klineInterval = schema.Interval1m
}

// SubscribeKlineSymbols adds symbols to kline subscription only
func (sm *SubscriptionManagerImpl) SubscribeKlineSymbols(symbols []string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var newlyAdded []string
	for _, symbol := range symbols {
		if _, exists := sm.klineSymbols[symbol]; !exists {
			sm.klineSymbols[symbol] = struct{}{}
			newlyAdded = append(newlyAdded, symbol)
		}
	}
	return newlyAdded
}

// SubscribeDepthSymbols adds symbols to depth subscription only
func (sm *SubscriptionManagerImpl) SubscribeDepthSymbols(symbols []string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var newlyAdded []string
	for _, symbol := range symbols {
		if _, exists := sm.depthSymbols[symbol]; !exists {
			sm.depthSymbols[symbol] = struct{}{}
			newlyAdded = append(newlyAdded, symbol)
		}
	}
	return newlyAdded
}

// GetKlineSymbols returns all currently subscribed kline symbols
func (sm *SubscriptionManagerImpl) GetKlineSymbols() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbols := make([]string, 0, len(sm.klineSymbols))
	for symbol := range sm.klineSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetDepthSymbols returns all currently subscribed depth symbols
func (sm *SubscriptionManagerImpl) GetDepthSymbols() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbols := make([]string, 0, len(sm.depthSymbols))
	for symbol := range sm.depthSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}
