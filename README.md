# ALL BY CURSOR
🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮CURSOR🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮🐮

# Exchange Connector

一个统一的加密货币交易所数据连接器，支持多交易所、多市场类型的数据获取和实时订阅。

## 功能特性

### 🚀 核心功能
- **多交易所支持**: Binance, OKX, Bybit, Gate, MEXC
- **多市场类型**: 现货、U本位合约、币本位合约
- **实时数据订阅**: WebSocket连接，支持ticker、K线、深度数据
- **REST API获取**: 支持实时调用REST API获取数据
- **加权价格计算**: 基于交易所权重的价格聚合
- **加权深度计算**: 基于买一卖一平均值的深度数据聚合
- **数据缓存**: 内存缓存，提高数据访问效率

### 📊 数据类型
- **Ticker数据**: 最新价格、成交量等信息
- **K线数据**: 支持1m、3m、5m、15m、30m、1h、4h、1d等时间间隔
- **深度数据**: 订单簿买卖盘数据

### 🔧 使用方式
- **WebSocket订阅**: 实时数据流，低延迟
- **REST API调用**: 按需获取数据，适合低频查询
- **SDK接口**: 简化的API接口，易于集成

## 快速开始

### 1. 安装依赖
```bash
go mod tidy
```

### 2. 基本使用

#### 使用SDK（推荐）
```go
package main

import (
    "context"
    "exchange-connector/pkg/sdk"
    "exchange-connector/pkg/types"
)

func main() {
    // 创建SDK实例
    sdk := sdk.NewExchangeConnectorSDK()
    
    // 添加交易所（需要实际的交易所实现）
    // sdk.AddExchange(binanceSpot, 1)
    
    // 创建上下文
    ctx := context.Background()
    
    // 订阅WebSocket数据
    symbols := []string{"BTCUSDT", "ETHUSDT"}
    sdk.SubscribeTicker(ctx, types.ExchangeBinance, types.MarketSpot, symbols)
    
    // 启动WebSocket连接
    sdk.StartWebSocket(ctx)
    
    // 读取订阅的数据
    if ticker, ok := sdk.WatchTicker(types.MarketSpot, "BTC", "USDT"); ok {
        fmt.Printf("BTC价格: %.2f\n", ticker.Price)
    }
    
    // 通过REST API获取数据
    if ticker, err := sdk.FetchTicker(ctx, types.MarketSpot, "BTC", "USDT"); err == nil {
        fmt.Printf("REST获取BTC价格: %.2f\n", ticker.Price)
    }
}
```

#### 直接使用Manager
```go
package main

import (
    "context"
    "exchange-connector/internal/manager"
    "exchange-connector/pkg/types"
)

func main() {
    // 创建管理器
    mgr := manager.NewManager()
    
    // 添加交易所
    // mgr.AddExchange(binanceSpot, 1)
    
    // 订阅数据
    ctx := context.Background()
    mgr.SubscribeTickers(ctx, types.ExchangeBinance, types.MarketSpot, []string{"BTCUSDT"})
    
    // 启动WebSocket
    mgr.StartWS(ctx)
    
    // 读取数据
    ticker, ok := mgr.WatchTicker(types.MarketSpot, "BTC", "USDT")
    if ok {
        fmt.Printf("价格: %.2f\n", ticker.Price)
    }
}
```

### 3. 运行示例
```bash
# 运行基础演示
go run cmd/demo/main.go

# 运行SDK演示
go run cmd/sdk_demo/main.go

# 运行权重计算演示
go run cmd/weight_demo/main.go
```

## API 参考

### SDK 接口

#### 数据订阅
```go
// 订阅ticker数据
SubscribeTicker(ctx context.Context, exchange types.ExchangeName, market types.MarketType, symbols []string) error

// 订阅K线数据
SubscribeKline(ctx context.Context, exchange types.ExchangeName, market types.MarketType, symbols []string, interval types.Interval) error

// 订阅深度数据
SubscribeDepth(ctx context.Context, exchange types.ExchangeName, market types.MarketType, symbols []string) error
```

#### WebSocket数据读取
```go
// 读取ticker数据
WatchTicker(market types.MarketType, base, quote string) (types.Ticker, bool)

// 读取K线数据
WatchKline(market types.MarketType, base, quote string, interval types.Interval) ([]types.Kline, bool)

// 读取深度数据
WatchDepth(market types.MarketType, base, quote string) (types.Depth, bool)
```

#### REST API数据获取
```go
// 获取ticker数据
FetchTicker(ctx context.Context, market types.MarketType, base, quote string) (types.Ticker, error)

// 获取K线数据
FetchKline(ctx context.Context, market types.MarketType, base, quote string, interval types.Interval, limit int) ([]types.Kline, error)

// 获取深度数据
FetchDepth(ctx context.Context, market types.MarketType, base, quote string, limit int) (types.Depth, error)
```



### 支持的市场类型
- `types.MarketSpot`: 现货市场
- `types.MarketFuturesUSDT`: U本位合约市场
- `types.MarketFuturesCoin`: 币本位合约市场

### 支持的时间间隔
- `types.Interval1m`: 1分钟
- `types.Interval3m`: 3分钟
- `types.Interval5m`: 5分钟
- `types.Interval15m`: 15分钟
- `types.Interval30m`: 30分钟
- `types.Interval1h`: 1小时
- `types.Interval4h`: 4小时
- `types.Interval1d`: 1天

## 架构设计

### 核心组件
1. **Manager**: 统一管理所有交易所连接和数据流
2. **Cache**: 内存缓存，存储订阅的数据
3. **Exchange**: 交易所接口，支持REST和WebSocket
4. **SDK**: 简化的API接口，便于用户使用

### 数据流
```
交易所API → Exchange → Manager → Cache → SDK → 用户应用
```

### 权重计算
系统支持基于交易所权重的价格聚合：
- 每个交易所可以设置权重
- 加权价格 = Σ(价格 × 权重) / Σ权重
- 支持实时计算和缓存

- **容错机制**: 某个交易所数据不可用时，其他交易所仍可参与计算

## 开发指南

### 添加新交易所
1. 实现 `interfaces.Exchange` 接口
2. 实现 `interfaces.RESTClient` 接口
3. 实现 `interfaces.WSConnector` 接口
4. 在Manager中注册交易所

### 扩展数据类型
1. 在 `pkg/types/types.go` 中定义新类型
2. 在Cache中添加相应的存储方法
3. 在Manager中添加相应的处理方法
4. 在SDK中暴露相应的接口

## 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/cache -v
go test ./internal/manager -v
```

## 许可证
MIT License

## 贡献
欢迎提交Issue和Pull Request！