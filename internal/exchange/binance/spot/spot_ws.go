package spot

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"exchange-connector/internal/cache"
	"exchange-connector/pkg/interfaces"
	"exchange-connector/pkg/logger"
	"exchange-connector/pkg/schema"
)

const (
	spotWSBase = "wss://stream.binance.com:9443/ws"

	// WebSocket channel names
	channelKline = "kline" //默认@1000ms, 支持@2000ms
	channelDepth = "depth" // 默认@1000ms, 可选@100ms

	// WebSocket event types
	eventKline = "kline"
	eventDepth = "depthUpdate"
)

// binanceSubscriptionMessage represents Binance WebSocket subscription message
type binanceSubscriptionMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// SpotWS implements WSConnector for Binance Spot.
type SpotWS struct {
	dialer *websocket.Dialer
	conn   interfaces.WSConn
	mu     sync.RWMutex
	cache  *cache.MemoryCache

	// rest client for snapshots
	rest interfaces.RESTClient

	// per-symbol local order books for incremental depth maintenance
	orderBooks map[string]*orderBook

	// context used in read loop (for REST calls during WS handling)
	readCtx context.Context

	// subscription manager
	subs interfaces.SubscriptionManager

	// internal control
	stopCh chan struct{}

	// connection health monitoring
	lastPingTime   time.Time
	lastPongTime   time.Time
	lastMessage    time.Time
	healthMu       sync.RWMutex
	isConnected    bool
	reconnectCount int
}

type orderBook struct {
	lastUpdateId int64
	bids         map[string]decimal.Decimal // price -> qty
	asks         map[string]decimal.Decimal // price -> qty
}

func NewSpotWS(c *cache.MemoryCache, rest interfaces.RESTClient) *SpotWS {
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
	}
	ws := &SpotWS{
		dialer:      d,
		cache:       c,
		subs:        cache.NewSubscriptionManager(),
		rest:        rest,
		orderBooks:  make(map[string]*orderBook),
		stopCh:      make(chan struct{}),
		isConnected: false,
	}
	return ws
}

func (s *SpotWS) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		logger.Info("Binance Spot WS 已连接，跳过连接")
		return nil
	}

	// Connect to Binance WebSocket endpoint
	u, _ := url.Parse(spotWSBase)
	logger.Info("Binance Spot WS 开始连接...")
	conn, _, err := s.dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		logger.Error("Binance WS 连接失败: %v", err)
		s.healthMu.Lock()
		s.isConnected = false
		s.healthMu.Unlock()
		return err
	}
	logger.Info("Binance Spot WS 连接成功")

	wsConn := interfaces.WSShim{Conn: conn}
	// 设置 ping-pong 处理器
	s.setupPingPongHandlers(wsConn)

	// 设置连接
	s.conn = wsConn

	// 更新连接状态和时间
	now := time.Now()
	s.healthMu.Lock()
	s.isConnected = true
	s.lastMessage = now
	s.lastPongTime = now
	s.healthMu.Unlock()

	// 重连时会自动订阅币对
	subscribedSymbols := s.subs.GetSubscribedSymbols()

	if len(subscribedSymbols) > 0 {
		logger.Info("Binance WS 重连 重新订阅币对: %v", subscribedSymbols)
		_ = s.applySubscriptions(ctx)
	}
	return nil
}

func (s *SpotWS) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新连接状态
	s.healthMu.Lock()
	s.isConnected = false
	s.healthMu.Unlock()

	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// SendMessage sends a message to WebSocket server
func (s *SpotWS) SendMessage(ctx context.Context, message interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.conn == nil {
		return errors.New("WebSocket not connected")
	}

	// Convert message to JSON for logging
	if jsonData, err := json.Marshal(message); err == nil {
		logger.Info("Binance Spot WS SendMessage: %s", string(jsonData))
	}
	if err := s.conn.WriteJSON(message); err != nil {
		logger.Error("Binance WS 发送消息失败: %v", err)
		return err
	}

	return nil
}

func (s *SpotWS) SubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 只订阅K线频道
	newlyAdded := s.subs.SubscribeKlineSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Binance WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("Binance WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 kline 订阅消息（固定1m）
	subMsg := s.buildKlineSubscriptionMessage(newlyAdded, "1m")
	return s.SendMessage(ctx, subMsg)
}

func (s *SpotWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 退订所有频道
	actuallyRemoved := s.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Binance WS 所有币对都未订阅 kline，跳过退订请求")
		return nil
	}

	logger.Info("Binance WS 退订 kline: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return s.resubscribe(ctx)
}

func (s *SpotWS) SubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 只订阅深度频道
	newlyAdded := s.subs.SubscribeDepthSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Binance WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("Binance WS 新增订阅 depth: %v", newlyAdded)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 depth 订阅消息
	subMsg := s.buildDepthSubscriptionMessage(newlyAdded)
	return s.SendMessage(ctx, subMsg)
}

func (s *SpotWS) UnsubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 退订所有频道
	actuallyRemoved := s.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Binance WS 所有币对都未订阅 depth，跳过退订请求")
		return nil
	}

	logger.Info("Binance WS 退订 depth: %v", actuallyRemoved)

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return s.resubscribe(ctx)
}

// applySubscriptions sends subscription messages to the WebSocket server
func (s *SpotWS) applySubscriptions(ctx context.Context) error {
	// Build subscription message
	subMsg := s.buildSubscriptionMessage()
	if subMsg == nil {
		logger.Info("Binance WS 无订阅")
		return nil
	}

	// Use unified SendMessage method
	return s.SendMessage(ctx, subMsg)
}

// buildKlineSubscriptionMessage builds kline subscription message
func (s *SpotWS) buildKlineSubscriptionMessage(symbols []string, interval schema.Interval) *binanceSubscriptionMessage {
	var streams []string
	for _, symbol := range symbols {
		streams = append(streams, fmt.Sprintf("%s@%s_%s", strings.ToLower(symbol), channelKline, interval))
	}

	return &binanceSubscriptionMessage{
		Method: "SUBSCRIBE",
		Params: streams,
	}
}

// buildDepthSubscriptionMessage builds depth subscription message
func (s *SpotWS) buildDepthSubscriptionMessage(symbols []string) *binanceSubscriptionMessage {
	var streams []string
	for _, symbol := range symbols {
		streams = append(streams, fmt.Sprintf("%s@%s", strings.ToLower(symbol), channelDepth))
	}

	return &binanceSubscriptionMessage{
		Method: "SUBSCRIBE",
		Params: streams,
	}
}

func (s *SpotWS) buildSubscriptionMessage() *binanceSubscriptionMessage {
	var streams []string

	// Get kline and depth symbols separately
	klineSymbols := s.subs.GetKlineSymbols()
	depthSymbols := s.subs.GetDepthSymbols()

	// Add kline streams for kline symbols
	for _, symbol := range klineSymbols {
		streams = append(streams, fmt.Sprintf("%s@%s_1m", strings.ToLower(symbol), channelKline))
	}

	// Add depth streams for depth symbols
	for _, symbol := range depthSymbols {
		streams = append(streams, fmt.Sprintf("%s@%s", strings.ToLower(symbol), channelDepth))
	}

	if len(streams) == 0 {
		return nil
	}

	return &binanceSubscriptionMessage{
		Method: "SUBSCRIBE",
		Params: streams,
	}
}

func (s *SpotWS) resubscribe(ctx context.Context) error {
	// 现在使用消息订阅，不需要重连
	return s.applySubscriptions(ctx)
}

func (s *SpotWS) StartReading(ctx context.Context) error {
	logger.Info("Binance WS 开始读取消息...")

	// 启动后台 goroutine 来运行读取循环
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Binance WS 上下文取消")
				return
			case <-s.stopCh:
				logger.Info("Binance WS 停止信号")
				return
			default:
			}

			s.mu.RLock()
			conn := s.conn
			s.mu.RUnlock()

			if conn == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			var msg json.RawMessage
			if err := conn.ReadJSON(&msg); err != nil {
				logger.Error("Binance WS 读取消息失败: %v", err)
				s.healthMu.Lock()
				s.isConnected = false
				s.healthMu.Unlock()
				s.attemptReconnect(ctx)
				continue
			}

			// 更新最后消息时间
			s.healthMu.Lock()
			s.lastMessage = time.Now()
			s.healthMu.Unlock()

			logger.Debug("Binance WS 收到消息: %s", string(msg))
			s.handleMessage(msg)
		}
	}()

	return nil
}

func (s *SpotWS) handleMessage(data json.RawMessage) {
	// 首先尝试解析为订阅确认消息
	var subResponse struct {
		Result interface{} `json:"result"`
		ID     interface{} `json:"id"` // 使用interface{}来处理null值
	}
	if err := json.Unmarshal(data, &subResponse); err == nil {
		// 检查是否为订阅确认消息：result为null且ID存在（可能为null或数字）
		// 同时确保这不是事件消息（没有e字段）
		if subResponse.Result == nil {
			// 再次检查是否包含事件字段，避免误判
			var eventCheck struct {
				E string `json:"e"` // Event type
			}
			if json.Unmarshal(data, &eventCheck) == nil && eventCheck.E == "" {
				logger.Info("Binance WS 收到订阅确认消息")
				return
			}
		}
	}

	// 直接解析事件类型，这是最高效的方式
	var event struct {
		E  string `json:"e"` // Event type
		Et int64  `json:"E"` // Event time
		S  string `json:"s"` // Symbol
	}
	if err := json.Unmarshal(data, &event); err == nil && event.E != "" && event.S != "" {
		symbol := strings.ToUpper(event.S)

		switch event.E {
		case eventKline:
			s.handleKline(symbol, data)
		case eventDepth:
			s.handleDepth(symbol, data)
		default:
			logger.Error("Binance WS 未知事件类型: %s", event.E)
		}
		return
	}

	logger.Error("Binance WS 无法解析消息: %s", string(data))
}

func (s *SpotWS) handleKline(symbol string, data json.RawMessage) {
	var klineData struct {
		E  string `json:"e"` // Event type (should be "kline")
		Et int64  `json:"E"` // Event time
		S  string `json:"s"` // Symbol
		K  struct {
			T   int64  `json:"t"` // Kline start time
			Tf  int64  `json:"T"` // Kline close time
			S   string `json:"s"` // Symbol
			I   string `json:"i"` // Interval
			F   int64  `json:"f"` // First trade ID
			L   int64  `json:"L"` // Last trade ID
			O   string `json:"o"` // Open price
			C   string `json:"c"` // Close price
			H   string `json:"h"` // High price
			Low string `json:"l"` // Low price
			V   string `json:"v"` // Base asset volume
			N   int64  `json:"n"` // Number of trades
			X   bool   `json:"x"` // Is this kline closed?
			Q   string `json:"q"` // Quote asset volume
			Vq  string `json:"V"` // Taker buy base asset volume
			Qq  string `json:"Q"` // Taker buy quote asset volume
			B   string `json:"B"` // Ignore
		} `json:"k"`
	}

	if err := json.Unmarshal(data, &klineData); err != nil {
		logger.Error("Binance WS 解析kline失败: %v", err)
		return
	}

	// Verify this is a kline event
	if klineData.E != eventKline {
		logger.Info("Binance WS 收到非kline事件: %s", klineData.E)
		return
	}

	// Convert string values to decimal
	open, _ := decimal.NewFromString(klineData.K.O)
	close, _ := decimal.NewFromString(klineData.K.C)
	high, _ := decimal.NewFromString(klineData.K.H)
	low, _ := decimal.NewFromString(klineData.K.Low)
	volume, _ := decimal.NewFromString(klineData.K.V)
	quoteVolume, _ := decimal.NewFromString(klineData.K.Q)

	// Convert time values
	startTime := klineData.K.T
	closeTime := klineData.K.Tf

	// Calculate AdaptVolume
	var timeForAdaptVolume int64
	if klineData.K.X {
		// 如果K线已完成，使用收盘时间
		timeForAdaptVolume = closeTime
	} else {
		// 如果K线未完成，使用事件时间
		timeForAdaptVolume = klineData.Et
	}

	// 1) 取时间的秒部分（0-59，含毫秒），保留三位小数，例如 10.333
	secWithinMinute := time.UnixMilli(timeForAdaptVolume).Second()
	msWithinSecond := timeForAdaptVolume % 1000
	secondsPart := float64(secWithinMinute) + float64(msWithinSecond)/1000.0
	secondsPart = math.Round(secondsPart*1000) / 1000

	// 2) AdaptVolume = Volume / secondsPart
	var adaptVolume decimal.Decimal
	if secondsPart > 0 {
		adaptVolume = volume.Div(decimal.NewFromFloat(secondsPart))
	} else {
		adaptVolume = volume
	}

	k := schema.Kline{
		Exchange:    schema.BINANCE,
		Market:      schema.SPOT,
		Symbol:      symbol,
		Interval:    schema.Interval(klineData.K.I),
		OpenTime:    time.UnixMilli(startTime),
		CloseTime:   time.UnixMilli(closeTime),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       close,
		Volume:      volume,
		QuoteVolume: quoteVolume,
		TradeNum:    klineData.K.N,
		IsFinal:     klineData.K.X,
		EventTime:   time.UnixMilli(klineData.Et),
		AdaptVolume: adaptVolume,
	}

	s.cache.SetKline(k)
}

func (s *SpotWS) handleDepth(symbol string, data json.RawMessage) {
	var depthData struct {
		E  string     `json:"e"` // Event type (should be "depthUpdate")
		Et int64      `json:"E"` // Event time
		S  string     `json:"s"` // Symbol
		U  int64      `json:"U"` // First update ID in event
		Ue int64      `json:"u"` // Final update ID in event
		B  [][]string `json:"b"` // Bids to be updated
		A  [][]string `json:"a"` // Asks to be updated
	}

	if err := json.Unmarshal(data, &depthData); err != nil {
		logger.Error("Binance WS 解析depth失败: %v", err)
		return
	}

	// Verify this is a depth update event
	if depthData.E != eventDepth {
		logger.Error("Binance WS 收到非depth事件: %s", depthData.E)
		return
	}

	// 确保本地Order Book已初始化（REST快照）
	if err := s.ensureOrderBookInitialized(symbol); err != nil {
		logger.Error("初始化本地深度失败(symbol=%s): %v", symbol, err)
		return
	}

	// 等待快照完全加载
	s.mu.RLock()
	ob := s.orderBooks[symbol]
	last := ob.lastUpdateId
	s.mu.RUnlock()

	// 如果lastUpdateId为0，说明快照可能还在加载中，等待下一个事件
	if last == 0 {
		return
	}

	// 简化的Binance深度更新规则：
	// 1. 如果 u < lastUpdateId，说明是过期事件，忽略
	// 2. 如果 U > lastUpdateId+1000，说明序列号差距过大，重建快照
	// 3. 其他情况都应用更新

	if depthData.Ue < last {
		// 过期事件，忽略
		return
	}

	// 检查序列号差距
	if depthData.U > last+1000 {
		// 序列号差距过大，重建快照
		logger.Warn("序列号差距过大，重建快照 symbol=%s, last=%d, U=%d, gap=%d", symbol, last, depthData.U, depthData.U-last)
		if err := s.reloadOrderBookSnapshot(symbol); err != nil {
			logger.Error("重建快照失败 symbol=%s: %v", symbol, err)
			return
		}
		// 重新获取lastUpdateId
		s.mu.RLock()
		ob = s.orderBooks[symbol]
		last = ob.lastUpdateId
		s.mu.RUnlock()
	}

	// 应用深度更新
	s.applyDepthEvent(symbol, depthData.B, depthData.A, depthData.Ue)

	// 输出到缓存（排序并裁剪）
	depthSnapshot := s.buildDepthFromOrderBook(symbol, depthData.Et)
	s.cache.SetDepth(depthSnapshot)
}

// ensureOrderBookInitialized loads REST snapshot if not present for symbol
func (s *SpotWS) ensureOrderBookInitialized(symbol string) error {
	s.mu.Lock()
	ob, ok := s.orderBooks[symbol]
	s.mu.Unlock()

	if ok && ob.lastUpdateId > 0 {
		// 已经初始化且有效
		return nil
	}

	// 需要加载或重新加载快照
	logger.Info("开始加载REST深度快照: symbol=%s", symbol)
	return s.reloadOrderBookSnapshot(symbol)
}

// reloadOrderBookSnapshot fetches REST snapshot and replaces local order book
func (s *SpotWS) reloadOrderBookSnapshot(symbol string) error {
	if s.rest == nil {
		return errors.New("REST client not available for depth snapshot")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d, err := s.rest.GetDepth(ctx, symbol, 100)
	if err != nil {
		logger.Error("获取REST深度快照失败: symbol=%s, error=%v", symbol, err)
		return err
	}

	ob := &orderBook{
		lastUpdateId: 0,
		bids:         make(map[string]decimal.Decimal),
		asks:         make(map[string]decimal.Decimal),
	}

	// 加载买单和卖单数据，确保价格精度一致（去除末尾0位）
	for _, lv := range d.Bids {
		// 使用价格字符串作为key，去除末尾0位保持精度一致
		priceStr := lv.Price.String() // 使用String()自动去除末尾0位
		ob.bids[priceStr] = lv.Quantity
	}
	for _, lv := range d.Asks {
		// 使用价格字符串作为key，去除末尾0位保持精度一致
		priceStr := lv.Price.String() // 使用String()自动去除末尾0位
		ob.asks[priceStr] = lv.Quantity
	}

	// 处理LastUpdateId
	if d.LastUpdateId != "" {
		if v, err := strconv.ParseInt(d.LastUpdateId, 10, 64); err == nil {
			ob.lastUpdateId = v
			logger.Info("已加载REST深度快照 symbol=%s lastUpdateId=%d, 买单%d档, 卖单%d档",
				symbol, ob.lastUpdateId, len(ob.bids), len(ob.asks))
		} else {
			logger.Error("解析LastUpdateId失败 symbol=%s, LastUpdateId=%s: %v", symbol, d.LastUpdateId, err)
			return fmt.Errorf("解析LastUpdateId失败: %v", err)
		}
	} else {
		logger.Error("REST深度快照缺少LastUpdateId symbol=%s", symbol)
		return errors.New("REST深度快照缺少LastUpdateId")
	}

	s.mu.Lock()
	s.orderBooks[symbol] = ob
	s.mu.Unlock()

	return nil
}

// applyDepthEvent applies incremental updates to local order book and advances lastUpdateId
func (s *SpotWS) applyDepthEvent(symbol string, bids [][]string, asks [][]string, newLast int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ob := s.orderBooks[symbol]
	if ob == nil {
		logger.Error("Order Book不存在: symbol=%s", symbol)
		return
	}

	// 应用买单更新，确保价格精度一致（去除末尾0位）
	for _, b := range bids {
		if len(b) < 2 {
			continue
		}
		// 关键修复：将原始价格字符串转换为decimal再转回string，确保精度一致
		priceDecimal, _ := decimal.NewFromString(b[0])
		priceStr := priceDecimal.String() // 这样确保与REST快照的价格格式一致
		qty, _ := decimal.NewFromString(b[1])
		if qty.IsZero() {
			delete(ob.bids, priceStr)
		} else {
			ob.bids[priceStr] = qty
		}
	}

	// 应用卖单更新，确保价格精度一致（去除末尾0位）
	for _, a := range asks {
		if len(a) < 2 {
			continue
		}
		// 关键修复：将原始价格字符串转换为decimal再转回string，确保精度一致
		priceDecimal, _ := decimal.NewFromString(a[0])
		priceStr := priceDecimal.String() // 这样确保与REST快照的价格格式一致
		qty, _ := decimal.NewFromString(a[1])
		if qty.IsZero() {
			delete(ob.asks, priceStr)
		} else {
			ob.asks[priceStr] = qty
		}
	}

	// 更新lastUpdateId
	ob.lastUpdateId = newLast

	// 清理超出100档的价格档位，保持内存使用合理
	s.cleanupOrderBookLevels(ob)
}

// cleanupOrderBookLevels 清理超出100档的价格档位，只保留最重要的档位
func (s *SpotWS) cleanupOrderBookLevels(ob *orderBook) {
	const maxLevels = 100

	// 如果买单档位超过100，清理多余的档位
	if len(ob.bids) > maxLevels {
		// 将价格转换为可排序的格式
		type priceLevel struct {
			priceStr string
			price    decimal.Decimal
		}

		bidLevels := make([]priceLevel, 0, len(ob.bids))
		for priceStr := range ob.bids {
			if price, err := decimal.NewFromString(priceStr); err == nil {
				bidLevels = append(bidLevels, priceLevel{priceStr: priceStr, price: price})
			}
		}

		// 按价格降序排序（买单价格越高越重要）
		sort.Slice(bidLevels, func(i, j int) bool {
			return bidLevels[i].price.GreaterThan(bidLevels[j].price)
		})

		// 清理超出100档的档位
		for i := maxLevels; i < len(bidLevels); i++ {
			delete(ob.bids, bidLevels[i].priceStr)
		}
	}

	// 如果卖单档位超过100，清理多余的档位
	if len(ob.asks) > maxLevels {
		// 将价格转换为可排序的格式
		type priceLevel struct {
			priceStr string
			price    decimal.Decimal
		}

		askLevels := make([]priceLevel, 0, len(ob.asks))
		for priceStr := range ob.asks {
			if price, err := decimal.NewFromString(priceStr); err == nil {
				askLevels = append(askLevels, priceLevel{priceStr: priceStr, price: price})
			}
		}

		// 按价格升序排序（卖单价格越低越重要）
		sort.Slice(askLevels, func(i, j int) bool {
			return askLevels[i].price.LessThan(askLevels[j].price)
		})

		// 清理超出100档的档位
		for i := maxLevels; i < len(askLevels); i++ {
			delete(ob.asks, askLevels[i].priceStr)
		}
	}
}

// buildDepthFromOrderBook converts local order book to schema.Depth with sorted levels
func (s *SpotWS) buildDepthFromOrderBook(symbol string, eventTimeMs int64) schema.Depth {
	s.mu.RLock()
	ob := s.orderBooks[symbol]
	s.mu.RUnlock()

	if ob == nil {
		return schema.Depth{
			Exchange:     schema.BINANCE,
			Market:       schema.SPOT,
			Symbol:       symbol,
			Bids:         []schema.PriceLevel{},
			Asks:         []schema.PriceLevel{},
			UpdatedAt:    time.UnixMilli(eventTimeMs),
			LastUpdateId: "0",
		}
	}

	bids := make([]schema.PriceLevel, 0, len(ob.bids))
	asks := make([]schema.PriceLevel, 0, len(ob.asks))

	// 收集买单和卖单，去除末尾0位保持精度一致
	for priceStr, q := range ob.bids {
		price, _ := decimal.NewFromString(priceStr) // 从字符串重新解析，去除末尾0位保持精度一致
		bids = append(bids, schema.PriceLevel{Price: price, Quantity: q})
	}
	for priceStr, q := range ob.asks {
		price, _ := decimal.NewFromString(priceStr) // 从字符串重新解析，去除末尾0位保持精度一致
		asks = append(asks, schema.PriceLevel{Price: price, Quantity: q})
	}

	// 排序：买单降序，卖单升序
	sort.Slice(bids, func(i, j int) bool { return bids[i].Price.GreaterThan(bids[j].Price) })
	sort.Slice(asks, func(i, j int) bool { return asks[i].Price.LessThan(asks[j].Price) })

	// 限制输出深度档位为前100档
	const maxLevels = 100
	if len(bids) > maxLevels {
		bids = bids[:maxLevels]
	}
	if len(asks) > maxLevels {
		asks = asks[:maxLevels]
	}

	return schema.Depth{
		Exchange:     schema.BINANCE,
		Market:       schema.SPOT,
		Symbol:       symbol,
		Bids:         bids,
		Asks:         asks,
		UpdatedAt:    time.UnixMilli(eventTimeMs),
		LastUpdateId: fmt.Sprintf("%d", ob.lastUpdateId),
	}
}

func strconvParseFloat(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

// setupPingPongHandlers 设置 ping-pong 处理器
func (s *SpotWS) setupPingPongHandlers(conn interfaces.WSConn) {
	// 设置接收到 ping 的处理器 - 自动回应 pong
	conn.SetPingHandler(func(appData string) error {
		logger.Info("Binance WS 收到 ping，发送 pong 响应")
		s.healthMu.Lock()
		s.lastMessage = time.Now()
		s.healthMu.Unlock()
		return conn.Pong([]byte(appData))
	})

	// 设置接收到 pong 的处理器
	conn.SetPongHandler(func(appData string) error {
		logger.Info("Binance WS 收到 pong 响应")
		s.healthMu.Lock()
		s.lastPongTime = time.Now()
		s.lastMessage = time.Now()
		s.healthMu.Unlock()
		return nil
	})
}

// HandlePing 处理接收到的 ping（由接口要求实现）
func (s *SpotWS) HandlePing(data []byte) error {
	logger.Debug("Binance WS HandlePing called")
	s.healthMu.Lock()
	s.lastMessage = time.Now()
	s.healthMu.Unlock()

	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return errors.New("WebSocket not connected")
	}

	return conn.Pong(data)
}

// SendPing 主动发送 ping（Binance 不允许主动 ping，所以空实现）
func (s *SpotWS) SendPing(ctx context.Context) error {
	// Binance WebSocket 不允许客户端主动发送 ping
	// 这里实现接口但不执行任何操作
	logger.Debug("Binance WS SendPing called - Binance 不支持客户端主动 ping，忽略")
	return nil
}

// StartHealthCheck 启动连接健康监控
func (s *SpotWS) StartHealthCheck(ctx context.Context) error {
	logger.Info("Binance WS 启动健康监控")

	go func() {
		ticker := time.NewTicker(5 * time.Second) // 每5秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("Binance WS 健康监控退出")
				return
			case <-s.stopCh:
				logger.Info("Binance WS 健康监控收到停止信号")
				return
			case <-ticker.C:
				s.checkConnectionHealth(ctx)
			}
		}
	}()

	return nil
}

// checkConnectionHealth 检查连接健康状态
func (s *SpotWS) checkConnectionHealth(ctx context.Context) {
	s.healthMu.RLock()
	isConnected := s.isConnected
	lastMsg := s.lastMessage
	reconnectCount := s.reconnectCount
	s.healthMu.RUnlock()

	if !isConnected {
		logger.Warn("Binance WS 连接断开，尝试重连")
		s.attemptReconnect(ctx)
		return
	}

	now := time.Now()
	timeSinceLastMsg := now.Sub(lastMsg)

	// 如果超过30秒没有收到任何消息，认为连接异常
	if timeSinceLastMsg > 30*time.Second {
		logger.Warn("Binance WS 长时间未收到消息 (%.2f秒)，重连次数: %d，尝试重连",
			timeSinceLastMsg.Seconds(), reconnectCount)
		s.attemptReconnect(ctx)
		return
	}

	// 如果超过10秒没有收到消息，发出警告
	if timeSinceLastMsg > 10*time.Second {
		logger.Warn("Binance WS 长时间未收到消息: %.2f秒", timeSinceLastMsg.Seconds())
	}
}

// attemptReconnect 尝试重新连接
func (s *SpotWS) attemptReconnect(ctx context.Context) {
	s.healthMu.Lock()
	s.reconnectCount++
	reconnectCount := s.reconnectCount
	s.healthMu.Unlock()

	logger.Warn("Binance WS 开始重连 (第%d次)", reconnectCount)

	// 关闭现有连接
	s.Close()

	// 等待一段时间再重连，避免过于频繁
	waitTime := time.Duration(reconnectCount) * time.Second
	if waitTime > 30*time.Second {
		waitTime = 30 * time.Second
	}

	logger.Warn("Binance WS 等待 %.0f 秒后重连", waitTime.Seconds())
	time.Sleep(waitTime)

	// 尝试重新连接
	if err := s.Connect(ctx); err != nil {
		logger.Error("Binance WS 重连失败: %v", err)
		return
	}

	logger.Info("Binance WS 重连成功")

	// 重连成功，重置计数器
	s.healthMu.Lock()
	s.reconnectCount = 0
	s.healthMu.Unlock()
}
