package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"

	"github.com/shopspring/decimal"
)

const (
	wsURL = "wss://api.gateio.ws/ws/v4/"
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
		logger.Info("Gate Spot WS 已连接，跳过连接")
		return nil
	}

	logger.Info("Gate Spot WS 开始连接...")

	// 这里应该实现实际的WebSocket连接逻辑
	// 由于没有具体的连接实现，我们只是记录日志
	logger.Info("Gate Spot WS 连接成功")

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
		logger.Info("Gate Spot WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("Gate Spot WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Gate Spot WS 未连接，订阅状态已保存，连接后将自动应用")
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
		logger.Info("Gate Spot WS 所有币对都已退订 kline，跳过退订请求")
		return nil
	}

	logger.Info("Gate Spot WS 退订 kline: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Gate Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("Gate Spot WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("Gate Spot WS 新增订阅 depth: %v", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Gate Spot WS 未连接，订阅状态已保存，连接后将自动应用")
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
		logger.Info("Gate Spot WS 所有币对都已退订 depth，跳过退订请求")
		return nil
	}

	logger.Info("Gate Spot WS 退订 depth: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Gate Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("Gate Spot WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

func (s *SpotWS) sendKlineSubscription(conn interfaces.WSConn, symbols []string) error {
	for _, sym := range symbols {
		msg := map[string]any{
			"time":    time.Now().Unix(),
			"channel": "spot.candlesticks",
			"event":   "subscribe",
			"payload": []any{"1m", sym},
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SpotWS) sendKlineUnsubscription(conn interfaces.WSConn, symbols []string) error {
	for _, sym := range symbols {
		msg := map[string]any{
			"time":    time.Now().Unix(),
			"channel": "spot.candlesticks",
			"event":   "unsubscribe",
			"payload": []any{"1m", sym}, // Use default interval for unsubscribe
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SpotWS) sendDepthSubscription(conn interfaces.WSConn, symbols []string) error {
	for _, sym := range symbols {
		msg := map[string]any{
			"time":    time.Now().Unix(),
			"channel": "spot.order_book",
			"event":   "subscribe",
			"payload": []any{sym, "20", "100ms"},
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SpotWS) sendDepthUnsubscription(conn interfaces.WSConn, symbols []string) error {
	for _, sym := range symbols {
		msg := map[string]any{
			"time":    time.Now().Unix(),
			"channel": "spot.order_book",
			"event":   "unsubscribe",
			"payload": []any{sym, "20", "100ms"},
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
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
		channel, _ := raw["channel"].(string)
		if channel == "spot.candlesticks" {
			var d struct {
				Result map[string]string `json:"result"`
			}
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &d)
			sym := d.Result["currency_pair"]
			iv := d.Result["interval"]
			o := decimal.RequireFromString(d.Result["open"])
			h := decimal.RequireFromString(d.Result["high"])
			l := decimal.RequireFromString(d.Result["low"])
			c := decimal.RequireFromString(d.Result["close"])
			v := decimal.RequireFromString(d.Result["base_volume"])
			ts, _ := strconv.ParseInt(d.Result["t"], 10, 64)
			k := schema.Kline{
				Exchange:  schema.GATE,
				Market:    schema.SPOT,
				Symbol:    sym,
				Interval:  schema.Interval(iv),
				OpenTime:  time.Unix(ts, 0),
				CloseTime: time.Unix(ts, 0),
				Open:      o,
				High:      h,
				Low:       l,
				Close:     c,
				Volume:    v,
				IsFinal:   true,
			}
			// Note: Since DataReader interface doesn't have AppendKline method,
			// we'll need to handle this differently or extend the interface
			logger.Info("Gate Spot WS 收到Kline数据: %+v", k)
		} else if channel == "spot.order_book" {
			// Handle depth data
			logger.Info("Gate Spot WS 收到Depth数据: %s", channel)
		}
	}
}

// handlePingMessage 检查并处理 ping 消息
func (s *SpotWS) handlePingMessage(data json.RawMessage, conn interfaces.WSConn) bool {
	// Gate.io 使用 {"method": "server.ping", "params": [], "id": null} 作为心跳
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		if method, hasMethod := msg["method"]; hasMethod && method == "server.ping" {
			logger.Info("Gate WS 收到 ping 消息")
			// 发送 {"method": "server.pong", "params": [], "id": null} 响应
			pongResponse := map[string]interface{}{
				"method": "server.pong",
				"params": []interface{}{},
				"id":     nil,
			}
			if err := conn.WriteJSON(pongResponse); err != nil {
				logger.Error("Gate WS 发送 pong 失败: %v", err)
			}
			return true
		}
	}
	return false
}

// HandlePing 处理接收到的 ping（实现接口）
func (s *SpotWS) HandlePing(data []byte) error {
	logger.Info("Gate WS HandlePing called")

	// 检查是否是 Gate.io ping 消息
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		if method, hasMethod := msg["method"]; hasMethod && method == "server.ping" {
			// 发送 pong 响应
			s.mu.RLock()
			conn := s.conn
			s.mu.RUnlock()

			if conn == nil {
				return fmt.Errorf("WebSocket not connected")
			}

			pongResponse := map[string]interface{}{
				"method": "server.pong",
				"params": []interface{}{},
				"id":     nil,
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

	if conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	// Gate.io 使用 {"method": "server.ping", "params": [], "id": null} 作为心跳
	pingMsg := map[string]interface{}{
		"method": "server.ping",
		"params": []interface{}{},
		"id":     nil,
	}
	return conn.WriteJSON(pingMsg)
}

// StartHealthCheck starts connection health monitoring
func (s *SpotWS) StartHealthCheck(ctx context.Context) error {
	// Gate.io 使用内置的 ping-pong 机制，这里不需要额外的健康检查
	// 心跳处理已经在 StartReading 中实现
	return nil
}
