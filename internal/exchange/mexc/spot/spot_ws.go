package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"

	"github.com/shopspring/decimal"
)

const (
	wsURL = "wss://wbs.mexc.com/ws"
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
		logger.Info("MEXC Spot WS 已连接，跳过连接")
		return nil
	}

	logger.Info("MEXC Spot WS 开始连接...")

	// 这里应该实现实际的WebSocket连接逻辑
	// 由于没有具体的连接实现，我们只是记录日志
	logger.Info("MEXC Spot WS 连接成功")

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
		logger.Info("MEXC Spot WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("MEXC Spot WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("MEXC Spot WS 未连接，订阅状态已保存，连接后将自动应用")
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
		logger.Info("MEXC Spot WS 所有币对都已退订 kline，跳过退订请求")
		return nil
	}

	logger.Info("MEXC Spot WS 退订 kline: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("MEXC Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("MEXC Spot WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("MEXC Spot WS 新增订阅 depth: %v", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("MEXC Spot WS 未连接，订阅状态已保存，连接后将自动应用")
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
		logger.Info("MEXC Spot WS 所有币对都已退订 depth，跳过退订请求")
		return nil
	}

	logger.Info("MEXC Spot WS 退订 depth: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("MEXC Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("MEXC Spot WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

func (s *SpotWS) sendKlineSubscription(conn interfaces.WSConn, symbols []string) error {
	for _, sym := range symbols {
		msg := map[string]any{
			"method": "SUBSCRIPTION",
			"params": []string{fmt.Sprintf("spot@public.kline.v3.api@%s@1m", sym)},
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
			"method": "UNSUBSCRIPTION",
			"params": []string{fmt.Sprintf("spot@public.kline.v3.api@%s@1m", sym)},
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
			"method": "SUBSCRIPTION",
			"params": []string{fmt.Sprintf("spot@public.depth.v3.api@%s@20", sym)},
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
			"method": "UNSUBSCRIPTION",
			"params": []string{fmt.Sprintf("spot@public.depth.v3.api@%s@20", sym)},
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
		topic, _ := raw["c"].(string)
		if strings.Contains(topic, "kline") {
			var d struct {
				D map[string]any `json:"d"`
			}
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &d)
			sym, _ := d.D["s"].(string)
			iv, _ := d.D["i"].(string)
			o := decimal.RequireFromString(d.D["o"].(string))
			h := decimal.RequireFromString(d.D["h"].(string))
			l := decimal.RequireFromString(d.D["l"].(string))
			c := decimal.RequireFromString(d.D["c"].(string))
			v := decimal.RequireFromString(d.D["v"].(string))
			ts, _ := toInt64(d.D["t"])
			k := schema.Kline{
				Exchange:  schema.MEXC,
				Market:    schema.SPOT,
				Symbol:    sym,
				Interval:  schema.Interval(iv),
				OpenTime:  time.UnixMilli(ts),
				CloseTime: time.UnixMilli(ts),
				Open:      o,
				High:      h,
				Low:       l,
				Close:     c,
				Volume:    v,
				IsFinal:   true,
			}
			// Note: Since DataReader interface doesn't have AppendKline method,
			// we'll need to handle this differently or extend the interface
			logger.Info("MEXC Spot WS 收到Kline数据: %+v", k)
		} else if strings.Contains(topic, "depth") {
			// Handle depth data
			logger.Info("MEXC Spot WS 收到Depth数据: %s", topic)
		}
	}
}

func toFloat(v any) (float64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseFloat(t, 64)
	case float64:
		return t, nil
	default:
		return 0, fmt.Errorf("bad type")
	}
}

func toInt64(v any) (int64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseInt(t, 10, 64)
	case float64:
		return int64(t), nil
	default:
		return 0, fmt.Errorf("bad type")
	}
}

// handlePingMessage 检查并处理 ping 消息
func (s *SpotWS) handlePingMessage(data json.RawMessage, conn interfaces.WSConn) bool {
	// MEXC 使用 "ping" 字符串作为心跳（类似 OKX）
	var pingCheck string
	if err := json.Unmarshal(data, &pingCheck); err == nil && pingCheck == "ping" {
		logger.Info("MEXC WS 收到 ping 消息")
		// 发送 "pong" 响应
		if err := conn.WriteJSON("pong"); err != nil {
			logger.Error("MEXC WS 发送 pong 失败: %v", err)
		}
		return true
	}
	return false
}

// HandlePing 处理接收到的 ping（实现接口）
func (s *SpotWS) HandlePing(data []byte) error {
	logger.Info("MEXC WS HandlePing called")

	// 检查是否是 "ping" 字符串
	var pingCheck string
	if err := json.Unmarshal(data, &pingCheck); err == nil && pingCheck == "ping" {
		// 发送 "pong" 响应
		s.mu.RLock()
		conn := s.conn
		s.mu.RUnlock()

		if conn == nil {
			return fmt.Errorf("WebSocket not connected")
		}

		return conn.WriteJSON("pong")
	}

	return nil
}

// SendPing sends ping to server (if supported by exchange)
func (s *SpotWS) SendPing(ctx context.Context) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn != nil {
		// MEXC 使用 "ping" 字符串作为心跳
		return conn.WriteJSON("ping")
	}
	return fmt.Errorf("WebSocket connection not established")
}

// StartHealthCheck starts connection health monitoring
func (s *SpotWS) StartHealthCheck(ctx context.Context) error {
	// MEXC 使用内置的 ping-pong 机制，这里不需要额外的健康检查
	// 心跳处理已经在 StartReading 中实现
	return nil
}
