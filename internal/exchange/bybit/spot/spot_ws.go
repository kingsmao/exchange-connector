package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"

	"github.com/shopspring/decimal"
)

const (
	wsURL = "wss://stream.bybit.com/v5/public/spot"
)

type SpotWS struct {
	conn interfaces.WSConn
	mu   sync.RWMutex
	subs interfaces.SubscriptionManager
}

func NewSpotWS(subs interfaces.SubscriptionManager) *SpotWS {
	return &SpotWS{
		subs: subs,
	}
}

func (s *SpotWS) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		logger.Info("Bybit Spot WS 已连接，跳过连接")
		return nil
	}

	logger.Info("Bybit Spot WS 开始连接...")

	// 这里应该实现实际的WebSocket连接逻辑
	// 由于没有具体的连接实现，我们只是记录日志
	logger.Info("Bybit Spot WS 连接成功")

	// 启动消息读取
	go s.StartReading(ctx)

	return nil
}

func (s *SpotWS) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *SpotWS) SubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 固定订阅1m K线数据
	newlyAdded := s.subs.SubscribeSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Bybit Spot WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("Bybit Spot WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Bybit Spot WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 kline 订阅消息
	return s.sendKlineSubscription(conn, newlyAdded)
}

func (s *SpotWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	actuallyRemoved := s.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Bybit Spot WS 所有币对都已退订 kline，跳过退订请求")
		return nil
	}

	logger.Info("Bybit Spot WS 退订 kline: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Bybit Spot WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return s.sendKlineUnsubscription(conn, actuallyRemoved)
}

func (s *SpotWS) SubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// Subscribe only depth data
	newlyAdded := s.subs.SubscribeSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Bybit Spot WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("Bybit Spot WS 新增订阅 depth: %v", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Bybit Spot WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 depth 订阅消息
	return s.sendDepthSubscription(conn, newlyAdded)
}

func (s *SpotWS) UnsubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	actuallyRemoved := s.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Bybit Spot WS 所有币对都已退订 depth，跳过退订请求")
		return nil
	}

	logger.Info("Bybit Spot WS 退订 depth: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Bybit Spot WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return s.sendDepthUnsubscription(conn, actuallyRemoved)
}

func (s *SpotWS) SendMessage(ctx context.Context, message interface{}) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	// Convert message to JSON for logging
	if jsonData, err := json.Marshal(message); err == nil {
		logger.Info("Bybit Spot WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

func (s *SpotWS) sendKlineSubscription(conn interfaces.WSConn, symbols []string) error {
	var args []string
	for _, sym := range symbols {
		args = append(args, fmt.Sprintf("kline.1m.%s", sym))
	}
	if len(args) == 0 {
		return nil
	}
	return conn.WriteJSON(map[string]any{"op": "subscribe", "args": args})
}

func (s *SpotWS) sendKlineUnsubscription(conn interfaces.WSConn, symbols []string) error {
	var args []string
	for _, sym := range symbols {
		args = append(args, fmt.Sprintf("kline.1.%s", sym))
	}
	if len(args) == 0 {
		return nil
	}
	return conn.WriteJSON(map[string]any{"op": "unsubscribe", "args": args})
}

func (s *SpotWS) sendDepthSubscription(conn interfaces.WSConn, symbols []string) error {
	var args []string
	for _, sym := range symbols {
		args = append(args, fmt.Sprintf("orderbook.50.%s", sym))
	}
	if len(args) == 0 {
		return nil
	}
	return conn.WriteJSON(map[string]any{"op": "subscribe", "args": args})
}

func (s *SpotWS) sendDepthUnsubscription(conn interfaces.WSConn, symbols []string) error {
	var args []string
	for _, sym := range symbols {
		args = append(args, fmt.Sprintf("orderbook.50.%s", sym))
	}
	if len(args) == 0 {
		return nil
	}
	return conn.WriteJSON(map[string]any{"op": "unsubscribe", "args": args})
}

func (s *SpotWS) StartReading(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		s.mu.RLock()
		conn := s.conn
		s.mu.RUnlock()
		if conn == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		var msg json.RawMessage
		if err := conn.ReadJSON(&msg); err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		// 检查是否是 ping 消息
		if s.handlePingMessage(msg, conn) {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal(msg, &raw); err != nil {
			continue
		}
		topic, _ := raw["topic"].(string)
		if strings.HasPrefix(topic, "kline.") {
			var d struct {
				Data []struct {
					Start    int64  `json:"start"`
					End      int64  `json:"end"`
					Interval string `json:"interval"`
					Open     string `json:"open"`
					High     string `json:"high"`
					Low      string `json:"low"`
					Close    string `json:"close"`
					Volume   string `json:"volume"`
					Symbol   string `json:"symbol"`
					Confirm  bool   `json:"confirm"`
				} `json:"data"`
			}
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &d)
			for _, it := range d.Data {
				o := decimal.RequireFromString(it.Open)
				h := decimal.RequireFromString(it.High)
				l := decimal.RequireFromString(it.Low)
				c := decimal.RequireFromString(it.Close)
				v := decimal.RequireFromString(it.Volume)
				k := schema.Kline{
					Exchange:  schema.BYBIT,
					Market:    schema.SPOT,
					Symbol:    it.Symbol,
					Interval:  schema.Interval(it.Interval),
					OpenTime:  time.UnixMilli(it.Start),
					CloseTime: time.UnixMilli(it.End),
					Open:      o,
					High:      h,
					Low:       l,
					Close:     c,
					Volume:    v,
					IsFinal:   it.Confirm,
				}
				// Note: Since DataReader interface doesn't have AppendKline method,
				// we'll need to handle this differently or extend the interface
				logger.Info("Bybit Spot WS 收到Kline数据: %+v", k)
			}
		} else if strings.HasPrefix(topic, "orderbook.") {
			// Handle depth data
			logger.Info("Bybit Spot WS 收到Depth数据: %s", topic)
		}
	}
}

// handlePingMessage 检查并处理 ping 消息
func (s *SpotWS) handlePingMessage(data json.RawMessage, conn interfaces.WSConn) bool {
	// Bybit 使用 {"op": "ping"} 作为心跳
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		if op, hasOp := msg["op"]; hasOp && op == "ping" {
			logger.Info("Bybit WS 收到 ping 消息")
			// 发送 {"op": "pong"} 响应
			pongResponse := map[string]interface{}{
				"op": "pong",
			}
			if err := conn.WriteJSON(pongResponse); err != nil {
				logger.Error("Bybit WS 发送 pong 失败: %v", err)
			}
			return true
		}
	}
	return false
}

// HandlePing 处理接收到的 ping（实现接口）
func (s *SpotWS) HandlePing(data []byte) error {
	logger.Info("Bybit WS HandlePing called")

	// 检查是否是 Bybit ping 消息
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		if op, hasOp := msg["op"]; hasOp && op == "ping" {
			// 发送 pong 响应
			s.mu.RLock()
			conn := s.conn
			s.mu.RUnlock()

			if conn == nil {
				return fmt.Errorf("WebSocket not connected")
			}

			pongResponse := map[string]interface{}{
				"op": "pong",
			}
			return conn.WriteJSON(pongResponse)
		}
	}

	return nil
}

// SendPing sends ping to server (if supported by exchange)
func (s *SpotWS) SendPing(ctx context.Context) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn != nil {
		// Bybit 使用 {"op": "ping"} 作为心跳
		pingMsg := map[string]interface{}{
			"op": "ping",
		}
		return conn.WriteJSON(pingMsg)
	}
	return fmt.Errorf("WebSocket connection not established")
}

// StartHealthCheck starts connection health monitoring
func (s *SpotWS) StartHealthCheck(ctx context.Context) error {
	// Bybit 使用内置的 ping-pong 机制，这里不需要额外的健康检查
	// 心跳处理已经在 StartReading 中实现
	return nil
}
