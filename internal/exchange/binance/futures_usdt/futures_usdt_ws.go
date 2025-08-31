package futures_usdt

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingsmao/exchange-connector/internal/cache"
	"github.com/kingsmao/exchange-connector/pkg/interfaces"
	"github.com/kingsmao/exchange-connector/pkg/logger"
	"github.com/kingsmao/exchange-connector/pkg/schema"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const (
	wsURL = "wss://fstream.binance.com/stream"

	channelKline = "kline"       // @250ms
	channelDepth = "depth@500ms" // 默认@250ms, 可选@100ms, @500ms (为什么不用@250? 因为盘口闪烁太频繁效果反而不好)

	// 使用全局配置的健康检查间隔
	pongWait  = 60 * time.Second
	writeWait = 10 * time.Second
)

type binanceSubscriptionMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int64    `json:"id"`
}

type orderBook struct {
	LastUpdateID  int64
	LastEventU    int64                      // 上一个event的u值，用于验证序列号连续性
	Bids          map[string]decimal.Decimal // price -> quantity
	Asks          map[string]decimal.Decimal // price -> quantity
	IsInitialized bool
}

type FuturesUSDTWS struct {
	conn               *websocket.Conn
	mu                 sync.RWMutex
	cache              *cache.MemoryCache
	subs               interfaces.SubscriptionManager
	rest               interfaces.RESTClient
	orderBooks         map[string]*orderBook
	ctx                context.Context
	cancel             context.CancelFunc
	healthCheckStarted bool

	// 重连相关
	reconnectCount int
	reconnectMu    sync.RWMutex
}

func NewFuturesUSDTWS(cache *cache.MemoryCache, subs interfaces.SubscriptionManager, rest interfaces.RESTClient) *FuturesUSDTWS {
	ctx, cancel := context.WithCancel(context.Background())
	return &FuturesUSDTWS{
		cache:      cache,
		subs:       subs,
		rest:       rest,
		orderBooks: make(map[string]*orderBook),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (f *FuturesUSDTWS) Connect(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.conn != nil {
		logger.Info("Binance Futures USDT WS 已连接，跳过连接")
		return nil
	}

	logger.Info("Binance Futures USDT WS 开始连接...")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		logger.Error("Binance Futures USDT WS 连接失败: %v", err)
		return err
	}

	f.conn = conn
	logger.Info("Binance Futures USDT WS 连接成功")

	// 设置连接参数
	conn.SetReadLimit(512 * 1024) // 512KB
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// 检查是否有现有订阅需要应用
	subscribedSymbols := f.subs.GetSubscribedSymbols()

	if len(subscribedSymbols) > 0 {
		_ = f.applySubscriptions(ctx)
	}
	return nil
}

func (f *FuturesUSDTWS) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.conn != nil {
		return f.conn.Close()
	}
	return nil
}

func (f *FuturesUSDTWS) SubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 只订阅K线频道
	newlyAdded := f.subs.SubscribeKlineSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Binance Futures USDT WS 所有币对都已订阅 kline，跳过订阅请求")
		return nil
	}

	logger.Info("Binance Futures USDT WS 新增订阅 kline: %v (固定1m)", newlyAdded)

	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance Futures USDT WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 kline 订阅消息（固定1m）
	subMsg := f.buildKlineSubscriptionMessage(newlyAdded, "1m")
	return f.SendMessage(ctx, subMsg)
}

func (f *FuturesUSDTWS) UnsubscribeKline(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	actuallyRemoved := f.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Binance Futures USDT WS 所有币对都已退订 kline，跳过退订请求")
		return nil
	}

	logger.Info("Binance Futures USDT WS 退订 kline: %v", actuallyRemoved)

	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance Futures USDT WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return f.resubscribe(ctx)
}

func (f *FuturesUSDTWS) SubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	// 只订阅深度频道
	newlyAdded := f.subs.SubscribeDepthSymbols(upperSymbols)
	if len(newlyAdded) == 0 {
		logger.Info("Binance Futures USDT WS 所有币对都已订阅 depth，跳过订阅请求")
		return nil
	}

	logger.Info("Binance Futures USDT WS 新增订阅 depth: %v", newlyAdded)

	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance Futures USDT WS 未连接，订阅状态已保存，连接后将自动应用")
		return nil
	}

	// 构建并发送 depth 订阅消息
	subMsg := f.buildDepthSubscriptionMessage(newlyAdded)
	return f.SendMessage(ctx, subMsg)
}

func (f *FuturesUSDTWS) UnsubscribeDepth(ctx context.Context, symbols []string) error {
	// Convert symbols to uppercase for consistency
	upperSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		upperSymbols[i] = strings.ToUpper(symbol)
	}

	actuallyRemoved := f.subs.UnsubscribeSymbols(upperSymbols)
	if len(actuallyRemoved) == 0 {
		logger.Info("Binance Futures USDT WS 所有币对都已退订 depth，跳过退订请求")
		return nil
	}

	logger.Info("Binance Futures USDT WS 退订 depth: %v", actuallyRemoved)

	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()
	if conn == nil {
		logger.Warn("Binance Futures USDT WS 未连接，退订状态已保存，连接后将自动应用")
		return nil
	}
	return f.resubscribe(ctx)
}

func (f *FuturesUSDTWS) SendMessage(ctx context.Context, message interface{}) error {
	f.mu.RLock()
	conn := f.conn
	f.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(writeWait))

	// Marshal message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		logger.Error("Binance Futures USDT WS 序列化消息失败: %v", err)
		return err
	}

	// Send message
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Error("Binance Futures USDT WS 发送消息失败: %v", err)
		return err
	}

	// Convert message to JSON for logging
	if jsonData, err := json.Marshal(message); err == nil {
		logger.Info("Binance Futures USDT WS SendMessage: %s", string(jsonData))
	}
	return nil
}

func (f *FuturesUSDTWS) applySubscriptions(ctx context.Context) error {
	// Build subscription message
	subMsg := f.buildSubscriptionMessage()
	if subMsg == nil {
		logger.Info("Binance Futures USDT WS 无订阅")
		return nil
	}

	// Use unified SendMessage method
	return f.SendMessage(ctx, subMsg)
}

func (f *FuturesUSDTWS) resubscribe(ctx context.Context) error {
	// Build subscription message
	subMsg := f.buildSubscriptionMessage()
	if subMsg == nil {
		logger.Info("Binance Futures USDT WS 无订阅")
		return nil
	}

	// Use unified SendMessage method
	return f.SendMessage(ctx, subMsg)
}

// buildSubscriptionMessage builds unified subscription message
func (f *FuturesUSDTWS) buildSubscriptionMessage() *binanceSubscriptionMessage {
	var streams []string

	// Get kline and depth symbols separately
	klineSymbols := f.subs.GetKlineSymbols()
	depthSymbols := f.subs.GetDepthSymbols()

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
		ID:     f.generateRandomID(),
	}
}

// buildKlineSubscriptionMessage builds kline subscription message
func (f *FuturesUSDTWS) buildKlineSubscriptionMessage(symbols []string, interval schema.Interval) *binanceSubscriptionMessage {
	var streams []string
	for _, symbol := range symbols {
		streams = append(streams, fmt.Sprintf("%s@%s_%s", strings.ToLower(symbol), channelKline, interval))
	}

	return &binanceSubscriptionMessage{
		Method: "SUBSCRIBE",
		Params: streams,
		ID:     f.generateRandomID(),
	}
}

// buildDepthSubscriptionMessage builds depth subscription message
func (f *FuturesUSDTWS) buildDepthSubscriptionMessage(symbols []string) *binanceSubscriptionMessage {
	var streams []string
	for _, symbol := range symbols {
		streams = append(streams, fmt.Sprintf("%s@%s", strings.ToLower(symbol), channelDepth))
	}

	return &binanceSubscriptionMessage{
		Method: "SUBSCRIBE",
		Params: streams,
		ID:     f.generateRandomID(),
	}
}

func (f *FuturesUSDTWS) generateRandomID() int64 {
	b := make([]byte, 8)
	rand.Read(b)
	// 确保最高位为0，避免负值
	b[0] &= 0x7F
	return int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
}

func (f *FuturesUSDTWS) handleRawMessage(message []byte) {
	// 打印原始消息用于调试
	logger.Debug("Binance Futures USDT WS 收到原始消息: %s", string(message))

	// 首先尝试解析为订阅确认消息
	var subResponse struct {
		Result interface{} `json:"result"`
		ID     int64       `json:"id"`
	}
	if err := json.Unmarshal(message, &subResponse); err == nil && subResponse.ID > 0 {
		logger.Info("Binance Futures USDT WS 收到订阅确认: ID=%d", subResponse.ID)
		return
	}

	var rawMsg map[string]interface{}
	if err := json.Unmarshal(message, &rawMsg); err != nil {
		logger.Error("Binance Futures USDT WS 解析原始消息失败: %v", err)
		return
	}

	// Check if it's a data stream
	if stream, ok := rawMsg["stream"].(string); ok {
		f.handleStreamData(stream, rawMsg["data"])
		return
	}

	logger.Error("Binance Futures USDT WS 收到未知消息: %s", string(message))
}

func (f *FuturesUSDTWS) handleStreamData(stream string, data interface{}) {
	// Parse stream name to get symbol and channel
	// 支持格式：btcusdt@depth@500ms, btcusdt@kline@1m
	parts := strings.Split(stream, "@")
	if len(parts) < 2 {
		logger.Error("Binance Futures USDT WS 无效的流名称: %s", stream)
		return
	}

	symbol := strings.ToUpper(parts[0])
	// 重建频道名称，包含时间间隔（如果有的话）
	channel := strings.Join(parts[1:], "@")

	// Convert data to JSON for parsing
	dataBytes, err := json.Marshal(data)
	if err != nil {
		logger.Error("Binance Futures USDT WS 序列化数据失败: %v", err)
		return
	}

	switch {
	case strings.HasPrefix(channel, "kline"):
		f.handleKline(symbol, dataBytes)
	case strings.HasPrefix(channel, "depth"):
		f.handleDepth(symbol, dataBytes)
	default:
		logger.Debug("Binance Futures USDT WS 未知频道: %s", channel)
	}
}

func (f *FuturesUSDTWS) handleKline(symbol string, data json.RawMessage) {
	var klineData struct {
		E  string `json:"e"` // Event type
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
		} `json:"k"`
	}

	if err := json.Unmarshal(data, &klineData); err != nil {
		logger.Error("Binance Futures USDT WS 解析kline失败: %v", err)
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

	// Calculate AdaptVolume (参考现货处理方式)
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
		Market:      schema.FUTURESUSDT,
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

	// Store kline data to cache
	f.cache.SetKline(k)
}

func (f *FuturesUSDTWS) handleDepth(symbol string, data json.RawMessage) {
	// 按照Binance官方文档实现OrderBook维护
	logger.Debug("Binance Futures USDT WS 处理 %s 深度数据", symbol)

	// 确保本地Order Book已初始化（REST快照）
	if err := f.ensureOrderBookInitialized(symbol); err != nil {
		logger.Error("初始化本地深度失败(symbol=%s): %v", symbol, err)
		return
	}

	// 等待快照完全加载
	f.mu.RLock()
	ob := f.orderBooks[symbol]
	last := ob.LastUpdateID
	f.mu.RUnlock()

	// 如果lastUpdateId为0，说明快照可能还在加载中，等待下一个事件
	if last == 0 {
		return
	}

	// 解析深度事件
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
		logger.Error("Binance Futures USDT WS 解析depth失败: %v", err)
		return
	}

	// Verify this is a depth update event
	if depthData.E != "depthUpdate" {
		logger.Error("Binance Futures USDT WS 收到非depth事件: %s", depthData.E)
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
		if err := f.reloadOrderBookSnapshot(symbol); err != nil {
			logger.Error("重建快照失败 symbol=%s: %v", symbol, err)
			return
		}
		// 重新获取lastUpdateId
		f.mu.RLock()
		ob = f.orderBooks[symbol]
		last = ob.LastUpdateID
		f.mu.RUnlock()
	}

	// 应用深度更新
	f.applyDepthEvent(symbol, depthData.B, depthData.A, depthData.Ue)

	// 输出到缓存（排序并裁剪）
	depthSnapshot := f.buildDepthFromOrderBook(symbol)
	if depthSnapshot != nil {
		f.cache.SetDepth(*depthSnapshot)
	} else {
		logger.Warn("Binance Futures USDT WS %s 构建深度数据失败", symbol)
	}
}

func (f *FuturesUSDTWS) ensureOrderBookInitialized(symbol string) error {
	f.mu.Lock()
	ob, ok := f.orderBooks[symbol]
	f.mu.Unlock()

	if ok && ob.LastUpdateID > 0 {
		// 已经初始化且有效
		return nil
	}

	// 需要加载或重新加载快照
	logger.Info("开始加载REST深度快照: symbol=%s", symbol)
	return f.reloadOrderBookSnapshot(symbol)
}

// reloadOrderBookSnapshot fetches REST snapshot and replaces local order book
func (f *FuturesUSDTWS) reloadOrderBookSnapshot(symbol string) error {
	if f.rest == nil {
		return errors.New("REST client not available for depth snapshot")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d, err := f.rest.GetDepth(ctx, symbol, 100)
	if err != nil {
		logger.Error("获取REST深度快照失败: symbol=%s, error=%v", symbol, err)
		return err
	}

	ob := &orderBook{
		LastUpdateID: 0,
		Bids:         make(map[string]decimal.Decimal),
		Asks:         make(map[string]decimal.Decimal),
	}

	// 加载买单和卖单数据，确保价格精度一致（去除末尾0位）
	for _, lv := range d.Bids {
		// 使用价格字符串作为key，去除末尾0位保持精度一致
		priceStr := lv.Price.String() // 使用String()自动去除末尾0位
		ob.Bids[priceStr] = lv.Quantity
	}
	for _, lv := range d.Asks {
		// 使用价格字符串作为key，去除末尾0位保持精度一致
		priceStr := lv.Price.String() // 使用String()自动去除末尾0位
		ob.Asks[priceStr] = lv.Quantity
	}

	// 处理LastUpdateId
	if d.LastUpdateId != "" {
		if v, err := strconv.ParseInt(d.LastUpdateId, 10, 64); err == nil {
			ob.LastUpdateID = v
			logger.Info("已加载REST深度快照 symbol=%s lastUpdateId=%d, 买单%d档, 卖单%d档",
				symbol, ob.LastUpdateID, len(ob.Bids), len(ob.Asks))
		} else {
			logger.Error("解析LastUpdateId失败 symbol=%s, LastUpdateId=%s: %v", symbol, d.LastUpdateId, err)
			return fmt.Errorf("解析LastUpdateId失败: %v", err)
		}
	} else {
		logger.Error("REST深度快照缺少LastUpdateId symbol=%s", symbol)
		return errors.New("REST深度快照缺少LastUpdateId")
	}

	f.mu.Lock()
	f.orderBooks[symbol] = ob
	f.mu.Unlock()

	return nil
}

// loadInitialDepthSnapshot loads initial depth snapshot from REST API
func (f *FuturesUSDTWS) loadInitialDepthSnapshot(ctx context.Context, symbol string) error {
	logger.Debug("Binance Futures USDT WS 检查REST客户端状态: %v", f.rest != nil)

	if f.rest == nil {
		logger.Error("Binance Futures USDT WS REST客户端不可用")
		return fmt.Errorf("REST client not available")
	}

	logger.Info("Binance Futures USDT WS 从REST API加载 %s 的初始深度快照", symbol)

	// 获取深度快照，使用100档深度
	depth, err := f.rest.GetDepth(ctx, symbol, 100)
	if err != nil {
		logger.Error("Binance Futures USDT WS 获取深度快照失败: %v", err)
		return err
	}

	// 解析LastUpdateID
	lastUpdateID, err := strconv.ParseInt(depth.LastUpdateId, 10, 64)
	if err != nil {
		logger.Error("Binance Futures USDT WS 解析LastUpdateID失败: %v", err)
		return err
	}

	f.mu.Lock()

	ob := f.orderBooks[symbol]
	if ob == nil {
		ob = &orderBook{
			Bids: make(map[string]decimal.Decimal),
			Asks: make(map[string]decimal.Decimal),
		}
		f.orderBooks[symbol] = ob
	}

	// 清空现有数据
	ob.Bids = make(map[string]decimal.Decimal)
	ob.Asks = make(map[string]decimal.Decimal)

	// 填充Bids
	for _, bid := range depth.Bids {
		ob.Bids[bid.Price.String()] = bid.Quantity
	}

	// 填充Asks
	for _, ask := range depth.Asks {
		ob.Asks[ask.Price.String()] = ask.Quantity
	}

	// 设置LastUpdateID
	ob.LastUpdateID = lastUpdateID

	logger.Info("Binance Futures USDT WS %s 深度快照加载完成，LastUpdateID: %d, Bids: %d, Asks: %d",
		symbol, lastUpdateID, len(ob.Bids), len(ob.Asks))

	// 释放写锁，避免死锁
	f.mu.Unlock()

	// 快照加载完成后，立即更新缓存
	depthSnapshot := f.buildDepthFromOrderBook(symbol)
	if depthSnapshot != nil {
		f.cache.SetDepth(*depthSnapshot)
		logger.Info("Binance Futures USDT WS %s 快照数据已更新到缓存", symbol)
	} else {
		logger.Warn("Binance Futures USDT WS %s 构建快照深度数据失败", symbol)
	}

	return nil
}

// applyDepthEvent applies incremental updates to local order book and advances lastUpdateId
func (f *FuturesUSDTWS) applyDepthEvent(symbol string, bids [][]string, asks [][]string, newLast int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ob := f.orderBooks[symbol]
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
			delete(ob.Bids, priceStr)
		} else {
			ob.Bids[priceStr] = qty
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
			delete(ob.Asks, priceStr)
		} else {
			ob.Asks[priceStr] = qty
		}
	}

	// 更新lastUpdateId
	ob.LastUpdateID = newLast

	// 清理超出100档的价格档位，保持内存使用合理
	f.cleanupOrderBookLevels(ob)
}

// cleanupOrderBookLevels 清理超出100档的价格档位，只保留最重要的档位
func (f *FuturesUSDTWS) cleanupOrderBookLevels(ob *orderBook) {
	// 清理买单，只保留前100档
	if len(ob.Bids) > 100 {
		// 将价格转换为decimal进行排序
		prices := make([]decimal.Decimal, 0, len(ob.Bids))
		for priceStr := range ob.Bids {
			if price, err := decimal.NewFromString(priceStr); err == nil {
				prices = append(prices, price)
			}
		}
		// 按价格降序排序（买单价格越高越优先）
		sort.Slice(prices, func(i, j int) bool {
			return prices[i].GreaterThan(prices[j])
		})
		// 只保留前100档
		keepPrices := prices[:100]
		keepPricesSet := make(map[string]bool)
		for _, price := range keepPrices {
			keepPricesSet[price.String()] = true
		}
		// 删除多余的档位
		for priceStr := range ob.Bids {
			if !keepPricesSet[priceStr] {
				delete(ob.Bids, priceStr)
			}
		}
	}

	// 清理卖单，只保留前100档
	if len(ob.Asks) > 100 {
		// 将价格转换为decimal进行排序
		prices := make([]decimal.Decimal, 0, len(ob.Asks))
		for priceStr := range ob.Asks {
			if price, err := decimal.NewFromString(priceStr); err == nil {
				prices = append(prices, price)
			}
		}
		// 按价格升序排序（卖单价格越低越优先）
		sort.Slice(prices, func(i, j int) bool {
			return prices[i].LessThan(prices[j])
		})
		// 只保留前100档
		keepPrices := prices[:100]
		keepPricesSet := make(map[string]bool)
		for _, price := range keepPrices {
			keepPricesSet[price.String()] = true
		}
		// 删除多余的档位
		for priceStr := range ob.Asks {
			if !keepPricesSet[priceStr] {
				delete(ob.Asks, priceStr)
			}
		}
	}
}

// applyDepthUpdate 按照Binance官方文档实现深度更新
func (f *FuturesUSDTWS) applyDepthUpdate(symbol string, event struct {
	EventType string     `json:"e"`  // Event type (e.g., "depthUpdate")
	E         int64      `json:"E"`  // Event time (number)
	T         int64      `json:"T"`  // Transaction time (number)
	S         string     `json:"s"`  // Symbol
	U         int64      `json:"U"`  // First update ID in event
	FinalU    int64      `json:"u"`  // Final update ID in event
	PrevU     int64      `json:"pu"` // Previous final update ID
	B         [][]string `json:"b"`  // Bids to update
	A         [][]string `json:"a"`  // Asks to update
}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	orderBook := f.orderBooks[symbol]
	if orderBook == nil {
		return fmt.Errorf("OrderBook不存在")
	}

	if orderBook.LastUpdateID == 0 {
		return fmt.Errorf("OrderBook尚未初始化")
	}

	// 步骤5: 验证更新序列号 (必须先验证这个)
	if event.FinalU < orderBook.LastUpdateID+1 {
		logger.Debug("Binance Futures USDT WS %s 跳过过期更新: u=%d < LastUpdateID+1=%d",
			symbol, event.FinalU, orderBook.LastUpdateID+1)
		return nil
	}

	// 步骤6: 验证序列号连续性 (pu应该等于上一个event的u)
	// 注意：第一次深度数据时LastEventU为0，这是正常的
	if orderBook.LastEventU != 0 && event.PrevU != orderBook.LastEventU {
		logger.Warn("Binance Futures USDT WS %s 序列号不连续，可能丢包: pu=%d, 上一个u=%d",
			symbol, event.PrevU, orderBook.LastEventU)
		// 需要重新初始化，但不要阻塞WebSocket读取
		go func() {
			if err := f.loadInitialDepthSnapshot(context.Background(), symbol); err != nil {
				logger.Error("Binance Futures USDT WS 重新获取 %s 深度快照失败: %v", symbol, err)
			}
		}()
		return fmt.Errorf("序列号不连续，需要重新初始化")
	}

	// 应用Bids更新
	for _, bid := range event.B {
		if len(bid) >= 2 {
			price := bid[0]
			quantity := bid[1]

			if qty, err := decimal.NewFromString(quantity); err == nil {
				if qty.IsZero() {
					// 步骤8: 挂单量为0，移除该价位
					delete(orderBook.Bids, price)
				} else {
					// 步骤7: 挂单量代表绝对值，不是相对变化
					orderBook.Bids[price] = qty
				}
			}
		}
	}

	// 应用Asks更新
	for _, ask := range event.A {
		if len(ask) >= 2 {
			price := ask[0]
			quantity := ask[1]

			if qty, err := decimal.NewFromString(quantity); err == nil {
				if qty.IsZero() {
					// 步骤8: 挂单量为0，移除该价位
					delete(orderBook.Asks, price)
				} else {
					// 步骤7: 挂单量代表绝对值，不是相对变化
					orderBook.Asks[price] = qty
				}
			}
		}
	}

	// 更新LastUpdateID和LastEventU
	orderBook.LastUpdateID = event.FinalU
	orderBook.LastEventU = event.FinalU

	logger.Debug("Binance Futures USDT WS %s 深度更新完成: LastUpdateID=%d, Bids=%d, Asks=%d",
		symbol, orderBook.LastUpdateID, len(orderBook.Bids), len(orderBook.Asks))

	return nil
}

func (f *FuturesUSDTWS) buildDepthFromOrderBook(symbol string) *schema.Depth {
	f.mu.RLock()
	defer f.mu.RUnlock()

	orderBook := f.orderBooks[symbol]
	if orderBook == nil {
		logger.Warn("Binance Futures USDT WS %s OrderBook不存在，无法构建深度数据", symbol)
		return nil
	}

	if orderBook.LastUpdateID == 0 {
		logger.Warn("Binance Futures USDT WS %s OrderBook未初始化，无法构建深度数据", symbol)
		return nil
	}

	// Build bids (sorted by price descending)
	var bids []schema.PriceLevel
	for price, quantity := range orderBook.Bids {
		if priceDec, err := decimal.NewFromString(price); err == nil {
			bids = append(bids, schema.PriceLevel{
				Price:    priceDec,
				Quantity: quantity,
			})
		} else {
			logger.Warn("Binance Futures USDT WS %s 价格解析失败: %s, error: %v", symbol, price, err)
		}
	}

	// Build asks (sorted by price ascending)
	var asks []schema.PriceLevel
	for price, quantity := range orderBook.Asks {
		if priceDec, err := decimal.NewFromString(price); err == nil {
			asks = append(asks, schema.PriceLevel{
				Price:    priceDec,
				Quantity: quantity,
			})
		} else {
			logger.Warn("Binance Futures USDT WS %s 价格解析失败: %s, error: %v", symbol, price, err)
		}
	}

	// Sort bids by price descending (highest bid first)
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	// Sort asks by price ascending (lowest ask first)
	sort.Slice(asks, func(i, j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})

	// 输出完整的100档深度数据
	var topBids []schema.PriceLevel
	if len(bids) > 100 {
		topBids = bids[:100] // 最多100档
	} else {
		topBids = bids // 实际有多少档就输出多少档
	}

	var topAsks []schema.PriceLevel
	if len(asks) > 100 {
		topAsks = asks[:100] // 最多100档
	} else {
		topAsks = asks // 实际有多少档就输出多少档
	}

	return &schema.Depth{
		Exchange:  schema.BINANCE,
		Market:    schema.FUTURESUSDT,
		Symbol:    symbol,
		Bids:      topBids,
		Asks:      topAsks,
		UpdatedAt: time.Now(),
	}
}

// StartHealthCheck starts connection health monitoring
func (f *FuturesUSDTWS) StartHealthCheck(ctx context.Context) error {
	ticker := time.NewTicker(schema.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Binance Futures USDT WS 健康检查停止")
			return nil
		case <-ticker.C:
			if err := f.SendPing(ctx); err != nil {
				logger.Warn("Binance Futures USDT WS ping失败: %v", err)
				// Try to reconnect
				f.reconnect()
			}
		}
	}
}

// StartReading starts read loop, handling heartbeats & reconnection internally
func (f *FuturesUSDTWS) StartReading(ctx context.Context) error {
	logger.Info("Binance Futures USDT WS 开始读取消息...")

	// 启动后台 goroutine 来运行读取循环
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Binance Futures USDT WS 上下文取消")
				return
			default:
			}

			f.mu.RLock()
			conn := f.conn
			f.mu.RUnlock()

			if conn == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(pongWait))

			// Read message
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error("Binance Futures USDT WS 读取消息失败: %v", err)
				}
				// 尝试重连
				f.reconnect()
				continue
			}

			// Handle message
			f.handleRawMessage(message)
		}
	}()

	// 启动健康检查
	if !f.healthCheckStarted {
		go f.StartHealthCheck(ctx)
		f.healthCheckStarted = true
	}

	return nil
}

func (f *FuturesUSDTWS) SendPing(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	conn := f.conn
	if conn == nil {
		return fmt.Errorf("connection not established")
	}

	conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.PingMessage, nil)
}

func (f *FuturesUSDTWS) HandlePing(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	conn := f.conn
	if conn == nil {
		return fmt.Errorf("connection not established")
	}

	conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.PongMessage, data)
}

func (f *FuturesUSDTWS) reconnect() {
	f.reconnectMu.Lock()
	f.reconnectCount++
	reconnectCount := f.reconnectCount
	f.reconnectMu.Unlock()

	logger.Warn("Binance Futures USDT WS 开始重连 (第%d次)", reconnectCount)

	f.mu.Lock()
	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}
	f.mu.Unlock()

	// 等待一段时间再重连，避免过于频繁
	// 前N次：1秒、2秒、3秒...N秒递增
	// 超过N次后：固定最大等待时间间隔
	var waitTime time.Duration
	if reconnectCount <= schema.ReconnectThreshold {
		waitTime = time.Duration(reconnectCount) * time.Second
	} else {
		waitTime = schema.MaxReconnectWaitTime
	}

	logger.Warn("Binance Futures USDT WS 等待 %.0f 秒后重连", waitTime.Seconds())
	time.Sleep(waitTime)

	// Try to reconnect
	if err := f.Connect(f.ctx); err != nil {
		logger.Error("Binance Futures USDT WS 重连失败: %v", err)
		// Schedule another reconnection attempt
		time.AfterFunc(30*time.Second, func() {
			f.reconnect()
		})
		return
	}

	logger.Info("Binance Futures USDT WS 重连成功")

	// 重连成功，重置计数器
	f.reconnectMu.Lock()
	f.reconnectCount = 0
	f.reconnectMu.Unlock()
}
