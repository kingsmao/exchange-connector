package interfaces

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"

	"github.com/kingsmao/exchange-connector/pkg/schema"
)

// SubscriptionManager manages subscription state for WebSocket connections.
type SubscriptionManager interface {
	// SubscribeSymbols adds symbols to all subscriptions (kline, depth, etc.), returns newly added symbols
	SubscribeSymbols(symbols []string) []string

	// UnsubscribeSymbols removes symbols from all subscriptions, returns actually removed symbols
	UnsubscribeSymbols(symbols []string) []string

	// GetSubscribedSymbols returns all currently subscribed symbols
	GetSubscribedSymbols() []string

	// SubscribeKlineSymbols adds symbols to kline subscription only
	SubscribeKlineSymbols(symbols []string) []string

	// SubscribeDepthSymbols adds symbols to depth subscription only
	SubscribeDepthSymbols(symbols []string) []string

	// GetKlineSymbols returns all currently subscribed kline symbols
	GetKlineSymbols() []string

	// GetDepthSymbols returns all currently subscribed depth symbols
	GetDepthSymbols() []string

	// ClearAll clears all subscriptions
	ClearAll()
}

// HTTPClient abstracts resty client for testability.
type HTTPClient interface {
	New() *resty.Client
}

// WSConn abstracts websocket Conn for testability.
type WSConn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Close() error
	// Ping sends a ping frame to server
	Ping(data []byte) error
	// Pong sends a pong frame to server
	Pong(data []byte) error
	// SetPingHandler sets the handler for received ping frames
	SetPingHandler(h func(appData string) error)
	// SetPongHandler sets the handler for received pong frames
	SetPongHandler(h func(appData string) error)
	// WriteMessage writes a message of the given type with the given payload
	WriteMessage(messageType int, data []byte) error
}

// WSConnector defines WebSocket subscription behaviors.
type WSConnector interface {
	Connect(ctx context.Context) error
	Close() error

	// SendMessage sends a message to WebSocket server
	SendMessage(ctx context.Context, message interface{}) error

	SubscribeKline(ctx context.Context, symbols []string) error
	UnsubscribeKline(ctx context.Context, symbols []string) error

	SubscribeDepth(ctx context.Context, symbols []string) error
	UnsubscribeDepth(ctx context.Context, symbols []string) error

	// StartReading starts read loop, handling heartbeats & reconnection internally.
	StartReading(ctx context.Context) error

	// Ping-pong and health check methods
	// HandlePing handles incoming ping from server
	HandlePing(data []byte) error
	// SendPing sends ping to server (if supported by exchange)
	SendPing(ctx context.Context) error
	// StartHealthCheck starts connection health monitoring
	StartHealthCheck(ctx context.Context) error
}

// RESTClient defines HTTP APIs to fetch data.
type RESTClient interface {
	GetDepth(ctx context.Context, symbol string, limit int) (schema.Depth, error)

	// GetExchangeInfo 获取交易规则和交易对信息
	GetExchangeInfo(ctx context.Context) (schema.ExchangeInfo, error)

	// Account/Orders (placeholders)
	// GetAccount(ctx context.Context) (any, error)
	// GetPositions(ctx context.Context) (any, error)
	// PlaceOrder(ctx context.Context, req any) (any, error)
}

// Exchange bundles market type and available clients.
type Exchange interface {
	Name() schema.ExchangeName
	Market() schema.MarketType
	REST() RESTClient
	WS() WSConnector
}

// WSShim adapts real *websocket.Conn to WSConn.
type WSShim struct{ *websocket.Conn }

func (w WSShim) WriteJSON(v any) error                       { return w.Conn.WriteJSON(v) }
func (w WSShim) ReadJSON(v any) error                        { return w.Conn.ReadJSON(v) }
func (w WSShim) SetReadDeadline(t time.Time) error           { return w.Conn.SetReadDeadline(t) }
func (w WSShim) SetWriteDeadline(t time.Time) error          { return w.Conn.SetWriteDeadline(t) }
func (w WSShim) Close() error                                { return w.Conn.Close() }
func (w WSShim) Ping(data []byte) error                      { return w.WriteMessage(websocket.PingMessage, data) }
func (w WSShim) Pong(data []byte) error                      { return w.WriteMessage(websocket.PongMessage, data) }
func (w WSShim) SetPingHandler(h func(appData string) error) { w.Conn.SetPingHandler(h) }
func (w WSShim) SetPongHandler(h func(appData string) error) { w.Conn.SetPongHandler(h) }
func (w WSShim) WriteMessage(messageType int, data []byte) error {
	return w.Conn.WriteMessage(messageType, data)
}
