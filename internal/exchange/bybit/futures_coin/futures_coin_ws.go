package futures_coin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"
)

const (
	BybitFuturesCoinWSBase = "wss://stream.bybit.com/v5/public/linear"
)

// FuturesCoinWS implements WSConnector for Bybit Coin-margined Futures.
type FuturesCoinWS struct {
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

func NewFuturesCoinWS(c *cache.MemoryCache) *FuturesCoinWS {
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
	}
	ws := &FuturesCoinWS{dialer: d, cache: c}
	ws.subs.tickers = make(map[string]struct{})
	ws.subs.kline = make(map[string]schema.Interval)
	ws.subs.depth = make(map[string]struct{})
	ws.stopCh = make(chan struct{})
	return ws
}

func (f *FuturesCoinWS) Connect(ctx context.Context) error {
	// TODO: Implement Bybit futures coin WS connection
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) Close() error {
	// TODO: Implement Bybit futures coin WS close
	return nil
}

func (f *FuturesCoinWS) SubscribeTickers(ctx context.Context, symbols []string) error {
	// TODO: Implement Bybit futures coin WS ticker subscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) UnsubscribeTickers(ctx context.Context, symbols []string) error {
	// TODO: Implement Bybit futures coin WS ticker unsubscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) SubscribeKline(ctx context.Context, symbols []string) error {
	// TODO: Implement Bybit futures coin WS kline subscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// TODO: Implement Bybit futures coin WS kline unsubscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) SubscribeDepth(ctx context.Context, symbols []string) error {
	// TODO: Implement Bybit futures coin WS depth subscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) UnsubscribeDepth(ctx context.Context, symbols []string) error {
	// TODO: Implement %!s(MISSING) futures coin WS depth unsubscription
	return errors.New("not implemented")
}

func (f *FuturesCoinWS) StartReading(ctx context.Context) error {
	// TODO: Implement Bybit futures coin WS start
	return errors.New("not implemented")
}

// SendMessage sends a message to WebSocket server
func (f *FuturesCoinWS) SendMessage(ctx context.Context, message interface{}) error {
	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		return errors.New("WebSocket connection not established")
	}
	// Convert message to JSON for logging
	if jsonData, err := json.Marshal(message); err == nil {
		logger.Info("Bybit Futures Coin WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

// HandlePing 处理接收到的 ping（空实现）
func (ws *FuturesCoinWS) HandlePing(data []byte) error {
	// TODO: 实现 Bybit 币本位期货 的 ping 处理
	return nil
}

// SendPing 主动发送 ping（空实现）
func (ws *FuturesCoinWS) SendPing(ctx context.Context) error {
	// TODO: 实现 Bybit 币本位期货 的主动 ping
	return nil
}

// StartHealthCheck 启动连接健康监控（空实现）
func (ws *FuturesCoinWS) StartHealthCheck(ctx context.Context) error {
	// TODO: 实现 Bybit 币本位期货 的健康检查
	return nil
}
