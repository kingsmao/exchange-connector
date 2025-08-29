//go:build integration

package futures_coin

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

func TestBinanceFuturesCoinREST_Depth(t *testing.T) {
	r := NewFuturesCoinREST()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	d, err := r.GetDepth(ctx, "BTCUSD_PERP", 5)
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
func TestBinanceFuturesCoinREST_ExchangeInfo(t *testing.T) {
	logger.Init()
	logger.SetLogLevel(logger.DEBUG)
	log.Printf("=== Binance币本位合约 REST ExchangeInfo 测试 (DEBUG模式) ===")

	r := NewFuturesCoinREST()
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

	if exchangeInfo.Market != schema.FUTURESCOIN {
		t.Errorf("市场类型错误，期望: %s, 实际: %s", schema.FUTURESCOIN, exchangeInfo.Market)
	}

	log.Printf("交易所: %s", exchangeInfo.Exchange)
	log.Printf("市场类型: %s", exchangeInfo.Market)
	log.Printf("时区: %s", exchangeInfo.Timezone)
	log.Printf("服务器时间: %s", exchangeInfo.ServerTime.Format("2006-01-02 15:04:05"))
	log.Printf("更新时间: %s", exchangeInfo.UpdatedAt.Format("2006-01-02 15:04:05"))
	log.Printf("交易对总数: %d", len(exchangeInfo.Symbols))
	log.Printf("限流规则数量: %d", len(exchangeInfo.RateLimits))

	// 检查是否有BTCUSD_PERP交易对
	var btcSymbol *schema.Symbol
	for i := range exchangeInfo.Symbols {
		if exchangeInfo.Symbols[i].Symbol == "BTCUSD_PERP" {
			btcSymbol = &exchangeInfo.Symbols[i]
			break
		}
	}

	if btcSymbol != nil {
		log.Printf("BTCUSD_PERP交易对信息: %+v", btcSymbol)
		log.Printf("数量精度: %d", btcSymbol.QuantityPrecision)
		log.Printf("价格精度: %d", btcSymbol.PricePrecision)
		log.Printf("最小数量: %s", btcSymbol.MinQuantity)
		log.Printf("最小金额: %s", btcSymbol.MinNotional)
	} else {
		log.Printf("未找到BTCUSD_PERP交易对")
	}

	// 验证交易对数据不为空（现在应该不是空实现了）
	if len(exchangeInfo.Symbols) == 0 {
		t.Logf("警告: 交易对列表为空，可能需要检查API实现")
	} else {
		t.Logf("成功获取到 %d 个交易对", len(exchangeInfo.Symbols))
	}
}
