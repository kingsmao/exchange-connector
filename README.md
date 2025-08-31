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

### 🔄 WebSocket重连机制
- **自动重连**: 连接断开时自动检测并重连
- **智能退避**: 前30次重连使用递增间隔（1秒到30秒），超过30次后固定30秒间隔
- **无限重连**: 永不放弃，持续尝试恢复连接
- **订阅恢复**: 重连成功后自动重新订阅之前的所有币对数据
- **健康监控**: 每1秒检查连接状态，30秒无消息自动重连（可配置）

## 快速开始

### 1. 安装依赖
```bash
go get github.com/kingsmao/exchange-connector
```

### 2. 基本使用

#### 使用SDK（推荐）
```go
package main

import (
    "context"
    "fmt"

    "github.com/kingsmao/exchange-connector/pkg/schema"
    "github.com/kingsmao/exchange-connector/pkg/sdk"
)

func main() {
	// 1. 创建SDK实例
	sdkInstance := sdk.NewSDK()

	// 2. 配置交易所（权重）
	if err := sdkInstance.AddExchange(sdk.ExchangeConfig{
		Name:   schema.BINANCE,
		Market: schema.SPOT,
		Weight: 3,
	}); err != nil {
		panic(err)
	}

	if err := sdkInstance.AddExchange(sdk.ExchangeConfig{
		Name:   schema.BINANCE,
		Market: schema.FUTURESUSDT,
		Weight: 1,
	}); err != nil {
		panic(err)
	}

	if err := sdkInstance.AddExchange(sdk.ExchangeConfig{
		Name:   schema.BINANCE,
		Market: schema.FUTURESCOIN,
		Weight: 1,
	}); err != nil {
		panic(err)
	}

	// 3. 配置币对（支持批量添加）
	allSymbols := []string{
		"BTC/USDT",      // 现货
		"ETH/USDT:USDT", // U本位合约
		"SOL/USD:SOL",   // 币本位合约
	}

	// 4. 使用便捷函数：添加币对并自动订阅WebSocket（一步完成）
	ctx := context.Background()
	if err := sdkInstance.AddSymbolsAndSubscribe(ctx, allSymbols); err != nil {
		panic(err)
	}

	// 创建定时器，每3秒执行一次
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// 创建退出信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			// 5. 读取数据（智能识别市场类型和交易所）
			if kline, ok := sdkInstance.WatchKline("BTC/USDT"); ok {
				fmt.Printf("现货BTC/USDT K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close)
			}
			if depth, ok := sdkInstance.WatchDepth("BTC/USDT"); ok {
				fmt.Printf("现货BTC/USDT 深度: 买单%d档, 卖单%d档, 买一:%s@%s, 卖一:%s@%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Bids[0].Quantity, depth.Asks[0].Price, depth.Asks[0].Quantity)
			}
			if kline, ok := sdkInstance.WatchKline("ETH/USDT:USDT"); ok {
				fmt.Printf("U本位合约ETH/USDT K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close)
			}
			if depth, ok := sdkInstance.WatchDepth("ETH/USDT:USDT"); ok {
				fmt.Printf("U本位合约ETH/USDT 深度: 买单%d档, 卖单%d档, 买一:%s@%s, 卖一:%s@%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Bids[0].Quantity, depth.Asks[0].Price, depth.Asks[0].Quantity)
			}
			if kline, ok := sdkInstance.WatchKline("SOL/USD:SOL"); ok {
				fmt.Printf("币本位合约SOL/USD K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close)
			}
			if depth, ok := sdkInstance.WatchDepth("SOL/USD:SOL"); ok {
				fmt.Printf("币本位合约SOL/USD 深度: 买单%d档, 卖单%d档, 买一:%s@%s, 卖一:%s@%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Bids[0].Quantity, depth.Asks[0].Price, depth.Asks[0].Quantity)
			}

			fmt.Println("---")
		}
	}

}
```



### 3. 运行示例
```bash
# 运行快速开始示例
cd quick_start
go run main.go
```

## API 参考

### SDK 接口

#### 交易所配置
```go
// 添加交易所配置
AddExchange(config ExchangeConfig) error

type ExchangeConfig struct {
    Name   schema.ExchangeName // 交易所名称
    Market schema.MarketType   // 市场类型
    Weight int                 // 权重
}
```

#### 币对配置和订阅
```go
// 批量添加币对并自动订阅（推荐）
AddSymbolsAndSubscribe(ctx context.Context, symbols []string) error

// 支持的币对格式
// 现货: "BTC/USDT"
// U本位合约: "BTC/USDT:USDT"  
// 币本位合约: "BTC/USD:BTC"
```

#### 数据读取
```go
// 读取K线数据（自动识别市场类型和交易所）
WatchKline(symbol string) (schema.Kline, bool)

// 读取深度数据（自动识别市场类型和交易所）
WatchDepth(symbol string) (schema.Depth, bool)

// 币对格式示例
// "BTC/USDT"      -> 现货市场
// "BTC/USDT:USDT" -> U本位合约
// "ETH/USD:ETH"   -> 币本位合约
```

#### 支持的交易所
- `schema.BINANCE` - Binance
- `schema.OKX` - OKX
- `schema.BYBIT` - Bybit
- `schema.GATE` - Gate
- `schema.MEXC` - MEXC



### 支持的市场类型
- `schema.SPOT` - 现货市场
- `schema.FUTURESUSDT` - U本位合约市场
- `schema.FUTURESCOIN` - 币本位合约市场

### 币对格式说明
- **现货**: `BTC/USDT` - 基础币种/计价币种
- **U本位合约**: `BTC/USDT:USDT` - 基础币种/计价币种:保证金币种
- **币本位合约**: `BTC/USD:BTC` - 基础币种/计价币种:保证金币种

### 自动市场类型识别
SDK会根据币对格式自动识别市场类型，无需手动指定：

#### 现货市场
```
BTC/USDT
 ↑   ↑
 │   └── 计价币种 (USDT)
 └────── 基础币种 (BTC)
```

#### U本位合约 (Quote = Margin)
```
BTC/USDT:USDT
 ↑   ↑    ↑
 │   │    └── 保证金币种 (USDT) ← 与计价币种相同
 │   └────── 计价币种 (USDT)
 └────────── 基础币种 (BTC)
```
**特点**: 保证金币种 = 计价币种，用户使用计价币种作为保证金

#### 币本位合约 (Base = Margin)  
```
BTC/USD:BTC
 ↑   ↑   ↑
 │   │   └── 保证金币种 (BTC) ← 与基础币种相同
 │   └────── 计价币种 (USD)
 └────────── 基础币种 (BTC)
```
**特点**: 保证金币种 = 基础币种，用户使用基础币种作为保证金

## 架构设计

### 核心组件
1. **SDK**: 高级API接口，提供简化的使用方式（用户使用）
2. **Manager**: 统一管理所有交易所连接和数据流（内部实现）
3. **Cache**: 内存缓存，存储订阅的数据（内部实现）
4. **Exchange**: 交易所接口，支持REST和WebSocket（内部实现）
5. **SubscriptionManager**: 统一管理所有频道的订阅状态（内部实现）

### 数据流
```
交易所API → Exchange → Manager → Cache → SDK → 用户应用
```

### 智能数据读取
- **自动交易所选择**: SDK内置交易所优先级（Binance → OKX → Bybit → Gate → MEXC）
- **自动市场类型识别**: 根据币对格式自动判断现货/合约类型
- **统一数据接口**: 使用标准币对格式，无需关心具体交易所实现

### 连接可靠性
- **自动重连**: WebSocket连接断开时自动重连，无需人工干预
- **订阅状态管理**: 维护所有币对的订阅状态，重连后自动恢复
- **健康检查**: 定期监控连接状态，及时发现问题并重连
- **错误隔离**: 单个交易所的问题不影响其他交易所的正常运行

### 系统配置
- **全局常量**: 健康检查间隔、重连阈值等配置集中在 `pkg/schema/constants.go` 中
- **易于维护**: 修改配置只需更新常量文件，无需修改多个交易所实现
- **统一标准**: 所有交易所使用相同的配置参数，保持一致性

### 批量订阅优化
- **分组订阅**: 按交易所和市场类型分组，批量订阅提高效率
- **增量订阅**: 支持在现有连接上添加新币对，无需重建连接
- **统一频道管理**: 币对订阅所有频道（K线、深度），统一管理订阅状态

## 开发指南

### 添加新交易所
1. 实现 `interfaces.Exchange` 接口
2. 实现 `interfaces.RESTClient` 接口  
3. 实现 `interfaces.WSConnector` 接口
4. 在Manager中注册交易所（内部实现）
5. 在SDK的`getDefaultExchangeOrder()`中添加交易所优先级

**WebSocket重连要求**:
- 实现 `StartHealthCheck()` 方法进行连接健康监控
- 实现自动重连逻辑，支持无限重试
- 重连成功后自动恢复之前的订阅状态
- 使用智能退避算法避免频繁重连
- 使用全局常量 `schema.HealthCheckInterval`、`schema.ReconnectThreshold` 等

### 扩展数据类型
1. 在 `pkg/schema/` 中定义新的数据结构
2. 在Cache中添加相应的存储方法
3. 在Manager中添加相应的处理方法
4. 在SDK中暴露相应的接口

### 自定义订阅逻辑
1. 修改 `pkg/sdk/sdk.go` 中的 `autoSubscribe` 方法
2. 调整交易所分组和批量订阅逻辑
3. 更新 `getDefaultExchangeOrder()` 中的交易所优先级

## 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/cache -v
go test ./pkg/sdk -v

# 运行快速开始示例
cd quick_start
go run main.go
```

## 许可证
MIT License

## 贡献
欢迎提交Issue和Pull Request！