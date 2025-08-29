//go:build integration

package spot

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kingsmao/exchange-connector/internal/manager"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

func TestOKXSpotREST_Depth(t *testing.T) {
	r := NewSpotREST()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	d, err := r.GetDepth(ctx, "BTC-USDT", 5)
	if err != nil {
		t.Fatalf("depth error: %v", err)
	}
	if len(d.Bids) == 0 && len(d.Asks) == 0 {
		t.Fatalf("empty orderbook")
	}
	log.Printf("OKX REST Depth: Bids=%d, Asks=%d, 最佳买价=%v, 最佳卖价=%v",
		len(d.Bids), len(d.Asks), d.Bids[0].Price, d.Asks[0].Price)
}

func TestOKXSpotWS_Kline(t *testing.T) {
	log.Printf("=== 开始 OKX WS 集成测试 ===")
	m := manager.NewManager()
	ex := NewSpotExchange(m.Cache())
	m.AddExchange(ex, 1) // 添加权重参数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Printf("启动 OKX WS 连接...")
	go func() { _ = m.StartWS(ctx) }()
	_ = m.SubscribeKline(ctx, schema.OKX, schema.SPOT, []string{"BTC/USDT"}, schema.Interval1m)

	time.Sleep(2 * time.Second)
	deadline := time.Now().Add(120 * time.Second)

	log.Printf("等待接收 OKX kline 数据...")
	for time.Now().Before(deadline) {
		if kl, ok := m.Cache().GetKline(schema.OKX, schema.SPOT, "BTC-USDT", schema.Interval1m); ok && len(kl) > 0 {
			log.Printf("收到 OKX kline: 数量=%d, 最新=%+v", len(kl), kl[len(kl)-1])
			break
		}
		log.Printf("等待 OKX kline 数据... 当前时间: %v", time.Now().Format("15:04:05"))
		time.Sleep(500 * time.Millisecond)
	}
	if kl, ok := m.Cache().GetKline(schema.OKX, schema.SPOT, "BTC-USDT", schema.Interval1m); !ok || len(kl) == 0 {
		log.Printf("未收到 OKX kline 数据")
		t.Fatalf("no kline received")
	} else {
		log.Printf("成功收到 OKX kline: 数量=%d, 最新=%+v", len(kl), kl[len(kl)-1])
	}

	log.Printf("=== OKX WS 集成测试完成 ===")
}
