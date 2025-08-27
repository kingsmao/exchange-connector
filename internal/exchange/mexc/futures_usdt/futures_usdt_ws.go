package futures_usdt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"
)

const (
	MexcFuturesUSDTWSBase = "wss://wbs.mexc.com/ws"
)

// FuturesUSDTWS implements WSConnector for Mexc USDT-margined Futures.
type FuturesUSDTWS struct {
	dialer *websocket.Dialer
	conn   interfaces.WSConn
	mu     sync.RWMutex
	cache  *cache.MemoryCache

	// subscription state
	subs struct {
		tickers map[string]struct{}
		kline   map[string]schema.Interval
		depth   map[string]struct{}
	}

	// internal control
	stopCh chan struct{}
}

func NewFuturesUSDTWS(c *cache.MemoryCache) *FuturesUSDTWS {
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
	}
	ws := &FuturesUSDTWS{dialer: d, cache: c}
	ws.subs.tickers = make(map[string]struct{})
	ws.subs.kline = make(map[string]schema.Interval)
	ws.subs.depth = make(map[string]struct{})
	ws.stopCh = make(chan struct{})
	return ws
}

func (f *FuturesUSDTWS) Connect(ctx context.Context) error {
	// TODO: Implement Mexc futures USDT WS connection
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) Close() error {
	// TODO: Implement Mexc futures USDT WS close
	return nil
}

func (f *FuturesUSDTWS) SubscribeTickers(ctx context.Context, symbols []string) error {
	// TODO: Implement Mexc futures USDT WS ticker subscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) UnsubscribeTickers(ctx context.Context, symbols []string) error {
	// TODO: Implement Mexc futures USDT WS ticker unsubscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) SubscribeKline(ctx context.Context, symbols []string) error {
	// TODO: Implement Mexc futures USDT WS kline subscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// TODO: Implement Mexc futures USDT WS kline unsubscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) SubscribeDepth(ctx context.Context, symbols []string) error {
	// TODO: Implement Mexc futures USDT WS depth subscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) UnsubscribeDepth(ctx context.Context, symbols []string) error {
	// TODO: Implement %!s(MISSING) futures USDT WS depth unsubscription
	return errors.New("not implemented")
}

func (f *FuturesUSDTWS) StartReading(ctx context.Context) error {
	// TODO: Implement Mexc futures USDT WS start
	return errors.New("not implemented")
}

// SendMessage sends a message to WebSocket server
func (f *FuturesUSDTWS) SendMessage(ctx context.Context, message interface{}) error {
	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		return errors.New("WebSocket connection not established")
	}
	// Convert message to JSON for logging
	if jsonData, err := json.Marshal(message); err == nil {
		logger.Info("MEXC Futures USDT WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

// HandlePing 处理接收到的 ping（空实现）
func (ws *FuturesUSDTWS) HandlePing(data []byte) error {
	// TODO: 实现 MEXC USDT 期货 的 ping 处理
	return nil
}

// SendPing 主动发送 ping（空实现）
func (ws *FuturesUSDTWS) SendPing(ctx context.Context) error {
	// TODO: 实现 MEXC USDT 期货 的主动 ping
	return nil
}

// StartHealthCheck 启动连接健康监控（空实现）
func (ws *FuturesUSDTWS) StartHealthCheck(ctx context.Context) error {
	// TODO: 实现 MEXC USDT 期货 的健康检查
	return nil
}
