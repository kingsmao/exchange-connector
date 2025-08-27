//go:build integration

package spot

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

func TestBinanceSpotWS_Kline(t *testing.T) {
	// 初始化日志系统并设置为DEBUG级别
	logger.Init()
	logger.SetLogLevel(logger.DEBUG)

	log.Printf("=== Binance WS Kline 测试 ===")
	m := manager.NewManager()
	ex := NewSpotExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅多个币对的K线数据
	//if err := m.SubscribeKline(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT", "ETHUSDT"}, schema.Interval1m); err != nil {
	//	t.Fatalf("订阅 K线 失败: %v", err)
	//}

	// 测试不同时间间隔的订阅
	if err := m.SubscribeKline(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT"}, schema.Interval1m); err != nil {
		t.Fatalf("订阅 K线 失败: %v", err)
	}

	log.Printf("WebSocket 连接已建立，正在持续接收 K线 数据...")
	log.Printf("按 Ctrl+C 停止测试")

	// 启动goroutine定期检查并打印内存中的kline数据
	go func() {
		ticker := time.NewTicker(1 * time.Second) // 每5秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 从缓存中读取kline数据
				if klines, exists := m.Cache().GetKline(schema.BINANCE, schema.SPOT, "BTCUSDT", schema.Interval1m); exists && len(klines) > 0 {
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

func TestBinanceSpotWS_Depth(t *testing.T) {
	// 初始化日志系统并设置为DEBUG级别
	logger.Init()
	logger.SetLogLevel(logger.INFO)

	log.Printf("=== Binance WS Depth 测试 ===")
	m := manager.NewManager()
	ex := NewSpotExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅多个币对的深度数据
	if err := m.SubscribeDepth(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT", "ETHUSDT"}); err != nil {
		t.Fatalf("订阅 Depth 失败: %v", err)
	}

	// 启动goroutine定期检查并打印内存中的depth数据
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond) // 每0.5秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				printDepthData(m.Cache(), "BTCUSDT")
				//printDepthData(m.Cache(), "ETHUSDT")
			}
		}
	}()

	select {} // 永久阻塞，直到手动停止
}

// printDepthData 打印指定币对的深度数据
func printDepthData(cache *cache.MemoryCache, symbol string) {
	depth, ok := cache.GetDepth(schema.BINANCE, schema.SPOT, symbol)
	if !ok {
		log.Printf("[%s] 暂无深度数据", symbol)
		return
	}

	//log.Printf("[%s] 深度数据 (LastUpdateId: %s, 更新时间: %s)",
	//	symbol, depth.LastUpdateId, depth.UpdatedAt.Format("15:04:05.000"))

	//
	// 打印买5卖5
	//log.Printf("[%s] 买8档:", symbol)
	//if len(depth.Bids) >= 8 {
	//	log.Printf("  8. %s@%s", depth.Bids[7].Price.String(), depth.Bids[7].Quantity.String())
	//} else {
	//	log.Printf("  8. 数据不足")
	//}

	//log.Printf("[%s] 买36档:", symbol)
	if len(depth.Asks) >= 36 {
		log.Printf("  18. %s@%s", depth.Asks[4].Price.String(), depth.Asks[4].Quantity.String())
	} else {
		log.Printf("  18. 数据不足")
	}

	//log.Printf("[%s] 卖5档:", symbol)
	//for i := 0; i < min(5, len(depth.Asks)); i++ {
	//	log.Printf("  %d. %s@%s", i+1, depth.Asks[i].Price.String(), depth.Asks[i].Quantity.String())
	//}
	//
	//// 打印买10卖10
	//if len(depth.Bids) >= 10 || len(depth.Asks) >= 10 {
	//	log.Printf("[%s] 买10档:", symbol)
	//	for i := 0; i < min(10, len(depth.Bids)); i++ {
	//		log.Printf("  %d. %s@%s", i+1, depth.Bids[i].Price.String(), depth.Bids[i].Quantity.String())
	//	}
	//
	//	log.Printf("[%s] 卖10档:", symbol)
	//	for i := 0; i < min(10, len(depth.Asks)); i++ {
	//		log.Printf("  %d. %s@%s", i+1, depth.Asks[i].Price.String(), depth.Asks[i].Quantity.String())
	//	}
	//}
	//
	//// 打印深度统计
	//log.Printf("[%s] 深度统计: 买单%d档, 卖单%d档", symbol, len(depth.Bids), len(depth.Asks))
}
