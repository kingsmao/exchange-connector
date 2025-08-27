# Binance交易所测试文件结构

## 概述

Binance交易所的测试文件已经重新组织，将REST API测试和WebSocket测试分开放置，便于独立运行和调试。

## 文件结构

### 现货市场 (Spot)
```
internal/exchange/binance/spot/
├── spot_exchange.go          # 交易所主文件
├── spot_rest.go             # REST API实现
├── spot_ws.go               # WebSocket实现
├── spot_rest_test.go        # REST API测试文件
└── spot_ws_test.go          # WebSocket测试文件
```

### U本位合约市场 (Futures USDT)
```
internal/exchange/binance/futures_usdt/
├── futures_usdt_exchange.go          # 交易所主文件
├── futures_usdt_rest.go             # REST API实现
├── futures_usdt_ws.go               # WebSocket实现
├── futures_usdt_rest_test.go        # REST API测试文件
└── futures_usdt_ws_test.go          # WebSocket测试文件
```

### 币本位合约市场 (Futures Coin)
```
internal/exchange/binance/futures_coin/
├── futures_coin_exchange.go          # 交易所主文件
├── futures_coin_rest.go             # REST API实现
├── futures_coin_ws.go               # WebSocket实现
├── futures_coin_rest_test.go        # REST API测试文件
└── futures_coin_ws_test.go          # WebSocket测试文件
```

## 测试文件说明

### REST API测试文件
- **文件名**: `*_rest_test.go`
- **功能**: 测试REST API接口
- **测试内容**:
  - Ticker数据获取
  - K线数据获取
  - 深度数据获取
- **运行方式**: `go test -tags=integration -run "Test*REST"`

### WebSocket测试文件
- **文件名**: `*_ws_test.go`
- **功能**: 测试WebSocket连接和数据订阅
- **测试内容**:
  - Ticker数据订阅
  - K线数据订阅
  - 深度数据订阅
- **运行方式**: `go test -tags=integration -run "Test*WS"`

## 运行测试

### 1. 运行所有Binance测试
```bash
./scripts/test_binance.sh
```

### 2. 运行特定市场的REST API测试
```bash
# 现货市场
go test -tags=integration ./internal/exchange/binance/spot -run "TestBinanceSpotREST" -v

# U本位合约市场
go test -tags=integration ./internal/exchange/binance/futures_usdt -run "TestBinanceFuturesUSDTREST" -v

# 币本位合约市场
go test -tags=integration ./internal/exchange/binance/futures_coin -run "TestBinanceFuturesCoinREST" -v
```

### 3. 运行特定市场的WebSocket测试
```bash
# 现货市场
go test -tags=integration ./internal/exchange/binance/spot -run "TestBinanceSpotWS" -v

# U本位合约市场
go test -tags=integration ./internal/exchange/binance/futures_usdt -run "TestBinanceFuturesUSDTWS" -v

# 币本位合约市场
go test -tags=integration ./internal/exchange/binance/futures_coin -run "TestBinanceFuturesCoinWS" -v
```

### 4. 运行特定测试
```bash
# 运行单个测试
go test -tags=integration ./internal/exchange/binance/spot -run "TestBinanceSpotREST_Ticker" -v

# 运行多个测试
go test -tags=integration ./internal/exchange/binance/spot -run "TestBinanceSpotREST" -v
```

## 测试注意事项

1. **集成测试标签**: 所有测试都需要使用 `-tags=integration` 标签
2. **超时设置**: 建议设置适当的超时时间，如 `-timeout=300s`
3. **网络依赖**: 测试需要网络连接，确保能够访问Binance API
4. **权重参数**: WebSocket测试中已添加权重参数 `m.AddExchange(ex, 1)`

## 调试建议

1. **REST API调试**: 先运行REST API测试，确保基本连接正常
2. **WebSocket调试**: 在REST API测试通过后，再运行WebSocket测试
3. **日志查看**: 测试会输出详细的日志信息，便于调试
4. **超时调整**: 根据网络情况调整测试超时时间

## 测试数据

### 现货市场测试数据
- **交易对**: BTCUSDT
- **时间间隔**: 1分钟
- **深度限制**: 5档

### U本位合约测试数据
- **交易对**: BTCUSDT
- **时间间隔**: 1分钟
- **深度限制**: 5档

### 币本位合约测试数据
- **交易对**: BTCUSD
- **时间间隔**: 1分钟
- **深度限制**: 5档
