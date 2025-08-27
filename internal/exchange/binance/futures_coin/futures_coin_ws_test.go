//go:build integration

package futures_coin

import (
	"context"
	"log"
	"testing"
	"time"

	"exchange-connector/internal/cache"
	"exchange-connector/internal/manager"
	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"
)

func TestBinanceFuturesCoinWS_Kline(t *testing.T) {
	// 初始化日志系统并设置为DEBUG级别
	logger.Init()
	logger.SetLogLevel(logger.INFO)

	log.Printf("=== Binance币本位合约 WS Kline 测试 ===")
	m := manager.NewManager()
	ex := NewFuturesCoinExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅K线数据
	if err := m.SubscribeKline(ctx, schema.BINANCE, schema.FUTURESCOIN, []string{"BTCUSD_PERP"}, schema.Interval1m); err != nil {
		t.Fatalf("订阅 K线 失败: %v", err)
	}

	log.Printf("WebSocket 连接已建立，正在持续接收 K线 数据...")
	log.Printf("按 Ctrl+C 停止测试")

	// 启动goroutine定期检查并打印内存中的kline数据
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // 每5秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 从缓存中读取kline数据
				if klines, exists := m.Cache().GetKline(schema.BINANCE, schema.FUTURESCOIN, "BTCUSD_PERP", schema.Interval1m); exists && len(klines) > 0 {
					latest := klines[len(klines)-1] // 获取最新的kline
					log.Printf("[%s] 价格:%s 数量:%s AdaptVolume:%s 时间:%s",
						latest.Symbol,
						latest.Close.String(),
						latest.Volume.String(),
						latest.AdaptVolume.String(),
						latest.EventTime.Format("15:04:05"))
				} else {
					// 更智能的状态显示
					if !exists {
						log.Printf("缓存中尚未存储K线数据...")
					} else if len(klines) == 0 {
						log.Printf("缓存中K线数据为空...")
					}
				}
			}
		}
	}()

	select {} // 永久阻塞，直到手动停止
}

func TestBinanceFuturesCoinWS_Depth(t *testing.T) {
	// 初始化日志系统并设置为INFO级别
	logger.Init()
	logger.SetLogLevel(logger.INFO)

	log.Printf("=== Binance币本位合约 WS Depth 测试 ===")
	m := manager.NewManager()
	ex := NewFuturesCoinExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅深度数据
	if err := m.SubscribeDepth(ctx, schema.BINANCE, schema.FUTURESCOIN, []string{"BTCUSD_PERP"}); err != nil {
		t.Fatalf("订阅 Depth 失败: %v", err)
	}

	// 启动goroutine定期检查并打印内存中的depth数据
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // 每1秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				printDepthData(m.Cache(), "BTCUSD_PERP")
			}
		}
	}()

	select {} // 永久阻塞，直到手动停止
}

// printDepthData 打印指定币对的深度数据
func printDepthData(cache *cache.MemoryCache, symbol string) {
	depth, ok := cache.GetDepth(schema.BINANCE, schema.FUTURESCOIN, symbol)
	if !ok {
		log.Printf("[%s] 暂无深度数据", symbol)
		return
	}

	// 打印第5档卖单
	if len(depth.Bids) >= 5 {
		log.Printf("[%s] 第5档卖单: %s            @%s", symbol, depth.Bids[4].Price.String(), depth.Bids[4].Quantity.String())
	} else {
		log.Printf("[%s] 卖单数据不足5档", symbol)
	}
}
