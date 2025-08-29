//go:build integration

package futures_usdt

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

func TestBinanceFuturesUSDTREST_Depth(t *testing.T) {
	logger.Init()
	logger.SetLogLevel(logger.DEBUG)
	log.Printf("=== Binance USDT合约 REST Depth 测试 (DEBUG模式) ===")

	r := NewFuturesUSDTREST()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	d, err := r.GetDepth(ctx, "BTCUSDT", 5)
	if err != nil {
		t.Fatalf("depth error: %v", err)
	}
	if len(d.Bids) == 0 && len(d.Asks) == 0 {
		t.Fatalf("empty orderbook")
	}
	log.Printf("REST Depth: Bids=%d, Asks=%d, 最佳买价=%v, 最佳卖价=%v",
		len(d.Bids), len(d.Asks), d.Bids[0].Price, d.Asks[0].Price)
}

// ExchangeInfo test
func TestBinanceFuturesUSDTREST_ExchangeInfo(t *testing.T) {
	logger.Init()
	logger.SetLogLevel(logger.DEBUG)
	log.Printf("=== Binance USDT合约 REST ExchangeInfo 测试 (DEBUG模式) ===")

	r := NewFuturesUSDTREST()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exchangeInfo, err := r.GetExchangeInfo(ctx)
	if err != nil {
		t.Fatalf("获取交易规则信息失败: %v", err)
	}

	// 验证基本信息
	if exchangeInfo.Exchange != schema.BINANCE {
		t.Errorf("交易所名称错误，期望: %s, 实际: %s", schema.BINANCE, exchangeInfo.Exchange)
	}

	if exchangeInfo.Market != schema.FUTURESUSDT {
		t.Errorf("市场类型错误，期望: %s, 实际: %s", schema.FUTURESUSDT, exchangeInfo.Market)
	}

	if len(exchangeInfo.Symbols) == 0 {
		t.Fatalf("交易对列表为空")
	}

	log.Printf("交易所: %s", exchangeInfo.Exchange)
	log.Printf("市场类型: %s", exchangeInfo.Market)
	log.Printf("时区: %s", exchangeInfo.Timezone)
	log.Printf("服务器时间: %s", exchangeInfo.ServerTime.Format("2006-01-02 15:04:05"))
	log.Printf("更新时间: %s", exchangeInfo.UpdatedAt.Format("2006-01-02 15:04:05"))
	log.Printf("交易对总数: %d", len(exchangeInfo.Symbols))
	log.Printf("限流规则数量: %d", len(exchangeInfo.RateLimits))

	// 查找并验证BTCUSDT交易对
	var btcusdtSymbol *schema.Symbol
	for i := range exchangeInfo.Symbols {
		if exchangeInfo.Symbols[i].Symbol == "BTCUSDT" {
			btcusdtSymbol = &exchangeInfo.Symbols[i]
			break
		}
	}

	if btcusdtSymbol == nil {
		t.Fatalf("未找到BTCUSDT交易对")
	}

	log.Printf("=== BTCUSDT 交易对详细信息 ===")
	log.Printf("交易对: %s", btcusdtSymbol.Symbol)
	log.Printf("基础币种: %s", btcusdtSymbol.Base)
	log.Printf("计价币种: %s", btcusdtSymbol.Quote)
	log.Printf("保证金币种: %s", btcusdtSymbol.Margin)
	log.Printf("数量精度: %d", btcusdtSymbol.QuantityPrecision)
	log.Printf("价格精度: %d", btcusdtSymbol.PricePrecision)
	log.Printf("最小下单数量: %s", btcusdtSymbol.MinQuantity)
	log.Printf("最大下单数量: %s", btcusdtSymbol.MaxQuantity)
	log.Printf("最小下单金额: %s", btcusdtSymbol.MinNotional)

	// 验证BTCUSDT的基本信息
	if btcusdtSymbol.Base != "BTC" {
		t.Errorf("BTCUSDT基础币种错误，期望: BTC, 实际: %s", btcusdtSymbol.Base)
	}

	if btcusdtSymbol.Quote != "USDT" {
		t.Errorf("BTCUSDT计价币种错误，期望: USDT, 实际: %s", btcusdtSymbol.Quote)
	}

	if btcusdtSymbol.Margin != "USDT" {
		t.Errorf("BTCUSDT保证金币种错误，期望: USDT, 实际: %s", btcusdtSymbol.Margin)
	}

	if btcusdtSymbol.MinQuantity == "" {
		t.Error("BTCUSDT最小下单数量为空")
	}

	if btcusdtSymbol.MinNotional == "" {
		log.Printf("注意: BTCUSDT最小下单金额为空，这可能是正常的")
	}

	// 显示前10个交易对的基本信息
	log.Printf("=== 前10个交易对列表 ===")
	limit := 10
	if len(exchangeInfo.Symbols) < limit {
		limit = len(exchangeInfo.Symbols)
	}

	for i := 0; i < limit; i++ {
		symbol := exchangeInfo.Symbols[i]
		log.Printf("%d. %s (%s/%s:%s) - 数量精度:%d, 价格精度:%d",
			i+1, symbol.Symbol, symbol.Base, symbol.Quote, symbol.Margin,
			symbol.QuantityPrecision, symbol.PricePrecision)
	}

	// 显示限流规则
	log.Printf("=== 限流规则 ===")
	for i, rateLimit := range exchangeInfo.RateLimits {
		log.Printf("%d. 类型:%s, 间隔:%s, 间隔数:%d, 限制:%d",
			i+1, rateLimit.RateLimitType, rateLimit.Interval,
			rateLimit.IntervalNum, rateLimit.Limit)
	}
}
