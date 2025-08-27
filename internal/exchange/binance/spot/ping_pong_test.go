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

func TestBinanceSpotWS_Connection(t *testing.T) {
	// 初始化日志系统并设置为INFO级别
	logger.Init()
	logger.SetLogLevel(logger.INFO)

	log.Printf("=== Binance WS 连接测试 ===")
	m := manager.NewManager()
	ex := NewSpotExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅一个币对用于测试（使用kline替代ticker）
	if err := m.SubscribeKline(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT"}, schema.Interval1m); err != nil {
		t.Fatalf("订阅 kline 失败: %v", err)
	}

	// 运行测试 - 观察连接健康监控和重连机制
	log.Printf("测试开始，观察连接健康状态...")
	log.Printf("可以通过断开网络连接来测试重连功能")
	log.Printf("按 Ctrl+C 停止测试")

	// 监控连接状态
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 打印连接状态信息
				ws := ex.WS().(*SpotWS)
				ws.healthMu.RLock()
				isConnected := ws.isConnected
				lastMessage := ws.lastMessage
				reconnectCount := ws.reconnectCount
				ws.healthMu.RUnlock()

				timeSinceLastMsg := time.Since(lastMessage)
				log.Printf("连接状态: %t, 最后消息: %.1f秒前, 重连次数: %d",
					isConnected, timeSinceLastMsg.Seconds(), reconnectCount)
			}
		}
	}()

	select {} // 永久阻塞，直到手动停止
}

func TestBinanceSpotWS_HealthCheck(t *testing.T) {
	// 初始化日志系统并设置为INFO级别
	logger.Init()
	logger.SetLogLevel(logger.INFO)

	log.Printf("=== Binance WS 健康检查测试 ===")
	m := manager.NewManager()
	ex := NewSpotExchange(m.Cache())
	m.AddExchange(ex, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := m.StartWS(ctx); err != nil {
		t.Fatalf("启动 WebSocket 失败: %v", err)
	}

	// 订阅数据（使用kline替代ticker）
	if err := m.SubscribeKline(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT"}, schema.Interval1m); err != nil {
		t.Fatalf("订阅 kline 失败: %v", err)
	}

	log.Printf("健康检查测试运行中，将在20秒后自动结束...")
	log.Printf("观察健康检查日志输出...")

	// 等待上下文超时
	<-ctx.Done()
	log.Printf("健康检查测试完成")
}

func TestBinanceSpotWS_PingMethods(t *testing.T) {
	// 初始化日志系统
	logger.Init()
	logger.SetLogLevel(logger.DEBUG)

	log.Printf("=== Binance WS Ping 方法测试 ===")
	cache := cache.NewMemoryCache()
	ws := NewSpotWS(cache, nil)

	ctx := context.Background()

	// 测试 SendPing 方法（应该是空实现）
	if err := ws.SendPing(ctx); err != nil {
		t.Errorf("SendPing 应该成功返回（空实现）: %v", err)
	}

	// 测试 HandlePing 方法（在未连接状态下应该返回错误）
	if err := ws.HandlePing([]byte("test")); err == nil {
		t.Error("HandlePing 在未连接状态下应该返回错误")
	}

	log.Printf("Ping 方法测试完成")
}
