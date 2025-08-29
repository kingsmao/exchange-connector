# Exchange Connector SDK 快速开始

这个示例展示了如何使用新的自动订阅SDK接口，实现配置化、自动化的交易所数据订阅。

## 新的使用方式

### 1. 传统方式 vs 新方式

#### 传统方式（手动订阅）
```go
// 需要手动一个个订阅
sdk.SubscribeKline(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT"}, schema.Interval1m)
sdk.SubscribeDepth(ctx, schema.BINANCE, schema.SPOT, []string{"BTCUSDT"})
sdk.SubscribeKline(ctx, schema.OKX, schema.FUTURESUSDT, []string{"BTCUSDT"}, schema.Interval1m)
// ... 更多手动订阅
```

#### 新方式（自动订阅）
```go
// 1. 配置交易所和权重
sdk.AddExchange(sdk.ExchangeConfig{
    Name:   schema.BINANCE,
    Market: schema.SPOT,
    Weight: 3,
})

// 2. 配置币对
sdk.AddSymbol(sdk.SymbolConfig{
    Base:   "BTC",
    Quote:  "USDT",
    Market: schema.SPOT,
})

// 3. 一键自动订阅
sdk.AutoSubscribe(ctx)
```

### 2. 核心概念

#### ExchangeConfig（交易所配置）
```go
type ExchangeConfig struct {
    Name   schema.ExchangeName // 交易所名称：BINANCE, OKX等
    Market schema.MarketType   // 市场类型：SPOT, FUTURESUSDT, FUTURESCOIN
    Weight int                 // 权重：用于后续的加权计算
}
```

#### SymbolConfig（币对配置）
```go
type SymbolConfig struct {
    Base   string              // 基础货币：BTC, ETH, SOL等
    Quote  string              // 计价货币：USDT, USDC等
    Market schema.MarketType   // 市场类型：必须与交易所配置匹配
}

// 支持多种批量添加方式：
// 1. AddSymbolsByFormat([]string{"BTC/USDT", "BTC/USDT:USDT"}) - 批量添加（自动识别市场类型）
// 2. AddSymbolsByExchange(exchange, []string{"BTC/USDT", "ETH/USDT"}) - 按交易所批量添加（自动识别市场类型）
// 3. AddSymbols([]SymbolConfig{...}) - 使用结构体数组批量添加
// 4. AddSymbol(SymbolConfig{...}) - 单独添加

// 🚀 便捷函数（一步完成添加币对和订阅）：
// 1. AddSymbolsAndSubscribe(ctx, []string{"BTC/USDT", "ETH/USDT"}) - 批量添加币对并订阅
// 2. AddSymbolsByExchangeAndSubscribe(ctx, exchange, []string{"BTC/USDT", "ETH/USDT"}) - 按交易所添加币对并订阅
// 3. AddSymbolAndSubscribe(ctx, SymbolConfig{...}) - 添加单个币对并订阅

// 币对格式说明（自动识别市场类型）：
// - 现货: "BTC/USDT" (基础币种/计价币种)
// - U本位合约: "BTC/USDT:USDT" (基础币种/计价币种:保证金币种)
// - 币本位合约: "BTC/USD:BTC" (基础币种/计价币种:保证金币种)
```

### 3. 使用步骤

#### 步骤1：初始化SDK
```go
sdk := sdk.NewSDK()
```

#### 步骤2：添加交易所配置
```go
// 添加Binance现货市场，权重为3
sdk.AddExchange(sdk.ExchangeConfig{
    Name:   schema.BINANCE,
    Market: schema.SPOT,
    Weight: 3,
})

// 添加OKX U本位市场，权重为1
sdk.AddExchange(sdk.ExchangeConfig{
    Name:   schema.OKX,
    Market: schema.FUTURESUSDT,
    Weight: 1,
})
```

#### 步骤3：添加币对配置（支持多种方式）

```go
// 方式1：批量添加币对（自动识别市场类型）
allSymbols := []string{
    "BTC/USDT",        // 现货
    "ETH/USDT",        // 现货
    "BNB/USDT",        // 现货
    "BTC/USDT:USDT",   // U本位合约
    "SOL/USDT:USDT",   // U本位合约
    "ETH/USDT:USDT",   // U本位合约
}
sdk.AddSymbolsByFormat(allSymbols)

// 方式2：按交易所批量添加币对（自动识别市场类型）
binanceSymbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT"}
sdk.AddSymbolsByExchange(schema.BINANCE, binanceSymbols)

// 方式3：单独添加特殊币对
sdk.AddSymbol(sdk.SymbolConfig{
    Base:   "ADA",
    Quote:  "USDT",
    Market: schema.SPOT,
})

// 方式4：使用结构体数组批量添加
configs := []sdk.SymbolConfig{
    {Base: "DOT", Quote: "USDT", Market: schema.SPOT},
    {Base: "LINK", Quote: "USDT", Market: schema.SPOT},
}
sdk.AddSymbols(configs)
```

#### 步骤4：自动订阅（两种方式）

**方式A：分步操作**
```go
ctx := context.Background()
if err := sdk.AutoSubscribe(ctx); err != nil {
    panic(err)
}
```

**方式B：使用便捷函数（推荐）**
```go
ctx := context.Background()

// 批量添加币对并订阅（一步完成）
if err := sdk.AddSymbolsAndSubscribe(ctx, allSymbols); err != nil {
    panic(err)
}

// 或者单独添加币对并订阅
if err := sdk.AddSymbolAndSubscribe(ctx, sdk.SymbolConfig{
    Base:   "ADA",
    Quote:  "USDT",
    Market: schema.SPOT,
}); err != nil {
    panic(err)
}
```

### 4. 自动订阅逻辑

当调用 `AutoSubscribe()` 时，SDK会自动：

1. **创建交易所实例**：根据配置创建对应的交易所实例
2. **启动WebSocket连接**：为所有交易所启动WebSocket连接
3. **自动订阅币对**：
   - 为每个币对订阅所有时间间隔的K线数据
   - 为每个币对订阅深度数据
   - 自动匹配市场类型，确保币对订阅到正确的交易所

### 5. 订阅结果示例

根据上面的配置，自动订阅结果如下：

```
Binance现货市场 (权重: 3):
├── BTCUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
├── ETHUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
├── BNBUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
└── ADAUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度

OKX U本位市场 (权重: 1):
├── BTCUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
├── SOLUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
└── ETHUSDT: K线(1m,5m,15m,30m,1h,4h,1d) + 深度
```

### 6. 优势

#### 配置化
- 通过配置对象管理交易所和币对
- 支持批量配置，一次配置多个
- 配置与逻辑分离，易于维护

#### 自动化
- 自动创建交易所实例
- 自动启动WebSocket连接
- 自动匹配和订阅币对
- 减少手动操作，降低出错概率

#### 扩展性
- 易于添加新的交易所
- 易于添加新的币对
- 支持不同的市场类型
- 权重配置为后续功能预留

### 7. 运行示例

```bash
cd quick_start
go run main.go
```

### 8. 注意事项

1. **市场类型匹配**：币对的市场类型必须与交易所配置匹配
2. **权重设置**：权重用于后续的加权计算，目前不影响订阅逻辑
3. **自动订阅**：调用 `AutoSubscribe()` 后，所有配置的交易所和币对都会被自动订阅
4. **错误处理**：如果某个交易所连接失败，其他交易所仍会继续工作

### 9. 便捷函数完整示例

#### 🚀 一步完成添加币对和订阅
```go
package main

import (
    "context"
    "fmt"
    "github.com/kingsmao/exchange-connector/pkg/sdk"
    "github.com/kingsmao/exchange-connector/pkg/schema"
)

func main() {
    // 1. 初始化SDK
    sdkInstance := sdk.NewSDK()

    // 2. 添加交易所配置
    sdkInstance.AddExchange(sdk.ExchangeConfig{
        Name:   schema.BINANCE,
        Market: schema.SPOT,
        Weight: 3,
    })

    sdkInstance.AddExchange(sdk.ExchangeConfig{
        Name:   schema.OKX,
        Market: schema.FUTURESUSDT,
        Weight: 1,
    })

    // 3. 使用便捷函数：添加币对并自动订阅（一步完成）
    ctx := context.Background()
    
    // 方式A：批量添加币对并订阅
    allSymbols := []string{
        "BTC/USDT",        // 现货
        "ETH/USDT",        // 现货
        "BNB/USDT",        // 现货
        "BTC/USDT:USDT",   // U本位合约
        "SOL/USDT:USDT",   // U本位合约
        "ETH/USDT:USDT",   // U本位合约
    }
    
    if err := sdkInstance.AddSymbolsAndSubscribe(ctx, allSymbols); err != nil {
        panic(err)
    }

    // 方式B：按交易所添加币对并订阅
    binanceSymbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT"}
    if err := sdkInstance.AddSymbolsByExchangeAndSubscribe(ctx, schema.BINANCE, binanceSymbols); err != nil {
        panic(err)
    }

    // 方式C：单独添加币对并订阅
    if err := sdkInstance.AddSymbolAndSubscribe(ctx, sdk.SymbolConfig{
        Base:   "ADA",
        Quote:  "USDT",
        Market: schema.SPOT,
    }); err != nil {
        panic(err)
    }

    fmt.Println("SDK初始化完成，开始接收数据...")
}
```

### 10. 后续功能

这个新的接口为以下功能预留了扩展空间：
- 加权数据聚合
- 多交易所数据对比
- 自动故障转移
- 动态配置更新
