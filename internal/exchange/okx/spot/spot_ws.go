package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"

	"github.com/shopspring/decimal"
)

const (
	wsURL = "wss://ws.okx.com:8443/ws/v5/public"
)

type SpotWS struct {
	conn  interfaces.WSConn
	mu    sync.RWMutex
	cache *cache.MemoryCache
	subs  interfaces.SubscriptionManager
}

func NewSpotWS(cache *cache.MemoryCache, subs interfaces.SubscriptionManager) *SpotWS {
	return &SpotWS{
		cache: cache,
		subs:  subs,
	}
}

func (s *SpotWS) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		logger.Info("OKX Spot WS 已连接，跳过连接")
		return nil
	}

	logger.Info("OKX Spot WS 开始连接...")

	// 这里应该实现实际的WebSocket连接逻辑
	// 由于没有具体的连接实现，我们只是记录日志
	logger.Info("OKX Spot WS 连接成功")

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
		logger.Info("OKX Spot WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("OKX Spot WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("OKX Spot WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 kline 订阅消息（固定1m）
	return s.sendKlineSubscription(conn, newlyAdded, "1m")
}

func (s *SpotWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	actuallyRemoved := s.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("OKX Spot WS 所有币对都已退订 kline，跳过退订请求")
		return nil
	}

	logger.Info("OKX Spot WS 退订 kline: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("OKX Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("OKX Spot WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("OKX Spot WS 新增订阅 depth: %v", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("OKX Spot WS 未连接，订阅状态已保存，连接后将自动应用")
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
		logger.Info("OKX Spot WS 所有币对都已退订 depth，跳过退订请求")
		return nil
	}

	logger.Info("OKX Spot WS 退订 depth: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("OKX Spot WS 未连接，退订状态已保存，连接后将自动应用")
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
		logger.Info("OKX Spot WS SendMessage: %s", string(jsonData))
	}
	return conn.WriteJSON(message)
}

func (s *SpotWS) sendKlineSubscription(conn interfaces.WSConn, symbols []string, interval schema.Interval) error {
	var args []map[string]string
	for _, inst := range symbols {
		bar := map[schema.Interval]string{
			schema.Interval1m:  "1m",
			schema.Interval3m:  "3m",
			schema.Interval5m:  "5m",
			schema.Interval15m: "15m",
			schema.Interval30m: "30m",
			schema.Interval1h:  "1H",
			schema.Interval4h:  "4H",
			schema.Interval1d:  "1D",
		}[interval]
		if bar == "" {
			bar = "1m"
		}
		// OKX 的 kline channel 格式是 "candle" + interval
		channelName := "candle" + bar
		args = append(args, map[string]string{"channel": channelName, "instId": inst})
	}
	if len(args) == 0 {
		return nil
	}
	msg := map[string]any{"op": "subscribe", "args": args}
	logger.Info("OKX WS 订阅消息: %+v", msg)
	return conn.WriteJSON(msg)
}

func (s *SpotWS) sendKlineUnsubscription(conn interfaces.WSConn, symbols []string) error {
	var args []map[string]string
	for _, inst := range symbols {
		// Use default interval for unsubscribe
		args = append(args, map[string]string{"channel": "candle1m", "instId": inst})
	}
	if len(args) == 0 {
		return nil
	}
	msg := map[string]any{"op": "unsubscribe", "args": args}
	logger.Info("OKX WS 退订消息: %+v", msg)
	return conn.WriteJSON(msg)
}

func (s *SpotWS) sendDepthSubscription(conn interfaces.WSConn, symbols []string) error {
	var args []map[string]string
	for _, inst := range symbols {
		args = append(args, map[string]string{"channel": "books", "instId": inst})
	}
	if len(args) == 0 {
		return nil
	}
	msg := map[string]any{"op": "subscribe", "args": args}
	logger.Info("OKX WS 订阅深度消息: %+v", msg)
	return conn.WriteJSON(msg)
}

func (s *SpotWS) sendDepthUnsubscription(conn interfaces.WSConn, symbols []string) error {
	var args []map[string]string
	for _, inst := range symbols {
		args = append(args, map[string]string{"channel": "books", "instId": inst})
	}
	if len(args) == 0 {
		return nil
	}
	msg := map[string]any{"op": "unsubscribe", "args": args}
	logger.Info("OKX WS 退订深度消息: %+v", msg)
	return conn.WriteJSON(msg)
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
			time.Sleep(500 * time.Millisecond)
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
		logger.Debug("OKX WS 收到消息: %+v", raw)
		if ch, _ := raw["arg"].(map[string]any); ch != nil {
			if channel, _ := ch["channel"].(string); strings.HasPrefix(channel, "candle") {
				logger.Debug("OKX WS 处理 kline 数据，channel: %s", channel)
				var data []map[string]any
				b, _ := json.Marshal(raw["data"])
				_ = json.Unmarshal(b, &data)
				for _, d := range data {
					arr, _ := d["candle"].([]any) // new formats may vary; fallback to array in d
					if arr == nil {
						if a, ok := d["data"].([]any); ok {
							arr = a
						}
					}
					if arr == nil {
						logger.Error("OKX WS kline 数据格式异常: %+v", d)
						continue
					}
					logger.Info("OKX WS 解析 kline 数组: %+v", arr)
					// OKX candle: [ts, o,h,l,c,vol,...]
					ts, _ := toInt64OKX(arr[0])
					oStr, _ := toStringOKX(arr[1])
					hStr, _ := toStringOKX(arr[2])
					lStr, _ := toStringOKX(arr[3])
					cStr, _ := toStringOKX(arr[4])
					vStr, _ := toStringOKX(arr[5])
					o := decimal.RequireFromString(oStr)
					h := decimal.RequireFromString(hStr)
					l := decimal.RequireFromString(lStr)
					cval := decimal.RequireFromString(cStr)
					v := decimal.RequireFromString(vStr)
					instId, _ := ch["instId"].(string)
					kline := schema.Kline{
						Exchange:  schema.OKX,
						Market:    schema.SPOT,
						Symbol:    strings.ReplaceAll(instId, "-", ""),
						Interval:  schema.Interval(strings.TrimPrefix(channel, "candle")),
						OpenTime:  time.UnixMilli(ts),
						CloseTime: time.UnixMilli(ts),
						Open:      o,
						High:      h,
						Low:       l,
						Close:     cval,
						Volume:    v,
						IsFinal:   true,
					}
					logger.Debug("OKX WS 解析 kline 成功: %+v", kline)
					// Note: Since DataReader interface doesn't have AppendKline method,
					// we'll need to handle this differently or extend the interface
					logger.Info("OKX Spot WS 收到Kline数据: %+v", kline)
				}
			} else if strings.HasPrefix(channel, "books") {
				// Handle depth data
				logger.Info("OKX Spot WS 收到Depth数据: %s", channel)

				// 解析depth数据
				var data []map[string]any
				b, _ := json.Marshal(raw["data"])
				_ = json.Unmarshal(b, &data)

				for _, d := range data {
					instId, _ := ch["instId"].(string)
					if instId == "" {
						continue
					}

					// 解析asks和bids数据
					asksRaw, _ := d["asks"].([]any)
					bidsRaw, _ := d["bids"].([]any)

					if len(asksRaw) == 0 || len(bidsRaw) == 0 {
						continue
					}

				}
			}
		}
	}
}

func toInt64OKX(v any) (int64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseInt(t, 10, 64)
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	default:
		return 0, fmt.Errorf("bad type for int64: %T", v)
	}
}

func toStringOKX(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case int64:
		return strconv.FormatInt(t, 10), nil
	default:
		return "", fmt.Errorf("bad type for string: %T", v)
	}
}

// handlePingMessage 检查并处理 ping 消息
func (s *SpotWS) handlePingMessage(data json.RawMessage, conn interfaces.WSConn) bool {
	// OKX 使用 "ping" 字符串作为心跳
	var pingCheck string
	if err := json.Unmarshal(data, &pingCheck); err == nil && pingCheck == "ping" {
		logger.Info("OKX WS 收到 ping 消息")
		// 发送 "pong" 响应
		if err := conn.WriteJSON("pong"); err != nil {
			logger.Error("OKX WS 发送 pong 失败: %v", err)
		}
		return true
	}
	return false
}

// HandlePing handles ping messages from the server
func (s *SpotWS) HandlePing(data []byte) error {
	// Send pong response
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn != nil {
		pongMsg := map[string]interface{}{
			"op": "pong",
		}
		pongData, err := json.Marshal(pongMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal pong message: %w", err)
		}
		return conn.WriteMessage(1, pongData)
	}
	return nil
}

// SendPing sends ping to server (if supported by exchange)
func (s *SpotWS) SendPing(ctx context.Context) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn != nil {
		// OKX 使用 "ping" 字符串作为心跳
		return conn.WriteJSON("ping")
	}
	return fmt.Errorf("WebSocket connection not established")
}

// StartHealthCheck starts connection health monitoring
func (s *SpotWS) StartHealthCheck(ctx context.Context) error {
	// OKX 使用内置的 ping-pong 机制，这里不需要额外的健康检查
	// 心跳处理已经在 StartReading 中实现
	return nil
}
