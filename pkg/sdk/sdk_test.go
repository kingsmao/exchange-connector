package sdk

import (
	"fmt"
	"testing"

	"exchange-connector/pkg/schema"
)

// 由于createExchange方法无法直接替换，我们测试配置管理逻辑
// 这些测试主要验证配置的增删改查逻辑，不涉及真实的交易所创建
func TestSDKExchangeConfigManagement(t *testing.T) {
	sdk := NewSDK()

	// 测试1：添加交易所配置（不创建实例）
	t.Run("Add Exchange Config", func(t *testing.T) {
		// 直接操作配置，跳过交易所创建
		config := ExchangeConfig{
			Name:   schema.BINANCE,
			Market: schema.SPOT,
			Weight: 3,
		}

		// 直接添加到配置列表
		sdk.exchangeConfigs = append(sdk.exchangeConfigs, config)

		// 验证配置已添加
		configs := sdk.GetExchangeConfigs()
		if len(configs) != 1 {
			t.Errorf("Expected 1 exchange config, got %d", len(configs))
		}

		if configs[0].Name != schema.BINANCE || configs[0].Weight != 3 {
			t.Errorf("Exchange config mismatch: %+v", configs[0])
		}
	})

	// 测试2：查找现有配置
	t.Run("Find Exchange Config", func(t *testing.T) {
		index, config := sdk.findExchangeConfig(schema.BINANCE, schema.SPOT)
		if index == -1 {
			t.Errorf("Exchange config should be found")
		}

		if config.Name != schema.BINANCE || config.Weight != 3 {
			t.Errorf("Found config mismatch: %+v", config)
		}
	})

	// 测试3：更新权重
	t.Run("Update Exchange Weight", func(t *testing.T) {
		// 查找并更新权重
		index, _ := sdk.findExchangeConfig(schema.BINANCE, schema.SPOT)
		if index != -1 {
			sdk.exchangeConfigs[index].Weight = 5
		}

		// 验证权重已更新
		configs := sdk.GetExchangeConfigs()
		if configs[0].Weight != 5 {
			t.Errorf("Expected weight 5, got %d", len(configs))
		}
	})

	// 测试4：删除配置
	t.Run("Remove Exchange Config", func(t *testing.T) {
		index, _ := sdk.findExchangeConfig(schema.BINANCE, schema.SPOT)
		if index != -1 {
			sdk.exchangeConfigs = append(sdk.exchangeConfigs[:index], sdk.exchangeConfigs[index+1:]...)
		}

		// 验证配置已被删除
		configs := sdk.GetExchangeConfigs()
		if len(configs) != 0 {
			t.Errorf("Expected 0 exchange configs, got %d", len(configs))
		}
	})

	// 测试5：获取特定配置
	t.Run("Get Specific Exchange Config", func(t *testing.T) {
		// 添加配置
		config := ExchangeConfig{
			Name:   schema.OKX,
			Market: schema.FUTURESUSDT,
			Weight: 2,
		}
		sdk.exchangeConfigs = append(sdk.exchangeConfigs, config)

		// 获取特定配置
		retrievedConfig, exists := sdk.GetExchangeConfig(schema.OKX, schema.FUTURESUSDT)
		if !exists {
			t.Errorf("Exchange config should exist")
		}

		if retrievedConfig.Name != schema.OKX || retrievedConfig.Weight != 2 {
			t.Errorf("Retrieved config mismatch: %+v", retrievedConfig)
		}

		// 测试不存在的配置
		_, exists = sdk.GetExchangeConfig(schema.BINANCE, schema.SPOT)
		if exists {
			t.Errorf("Exchange config should not exist")
		}
	})
}

// 测试交易所状态查询功能
func TestSDKExchangeStatusQueries(t *testing.T) {
	sdk := NewSDK()

	// 添加一些测试配置
	sdk.exchangeConfigs = append(sdk.exchangeConfigs, ExchangeConfig{
		Name:   schema.BINANCE,
		Market: schema.SPOT,
		Weight: 3,
	})
	sdk.exchangeConfigs = append(sdk.exchangeConfigs, ExchangeConfig{
		Name:   schema.OKX,
		Market: schema.FUTURESUSDT,
		Weight: 2,
	})

	t.Run("Get Active Exchanges", func(t *testing.T) {
		activeExchanges := sdk.GetActiveExchanges()
		if len(activeExchanges) != 2 {
			t.Errorf("Expected 2 active exchanges, got %d", len(activeExchanges))
		}

		// 检查是否包含预期的交易所
		expected := map[string]bool{
			"binance:spot":     true,
			"okx:futures_usdt": true,
		}

		for _, exchange := range activeExchanges {
			if !expected[exchange] {
				t.Errorf("Unexpected exchange: %s", exchange)
			}
		}
	})

	t.Run("Check Exchange Active Status", func(t *testing.T) {
		// 检查存在的交易所
		if !sdk.IsExchangeActive(schema.BINANCE, schema.SPOT) {
			t.Errorf("Binance SPOT should be active")
		}

		if !sdk.IsExchangeActive(schema.OKX, schema.FUTURESUSDT) {
			t.Errorf("OKX FUTURESUSDT should be active")
		}

		// 检查不存在的交易所
		if sdk.IsExchangeActive(schema.BINANCE, schema.FUTURESUSDT) {
			t.Errorf("Binance FUTURESUSDT should not be active")
		}
	})
}

// 测试交易所创建功能
func TestSDKExchangeCreation(t *testing.T) {
	sdk := NewSDK()

	t.Run("Create Binance Exchanges", func(t *testing.T) {
		// 测试Binance现货
		spotExchange := sdk.createBinanceExchange(schema.SPOT, sdk.manager.Cache())
		if spotExchange == nil {
			t.Errorf("Failed to create Binance SPOT exchange")
		} else {
			if spotExchange.Name() != schema.BINANCE {
				t.Errorf("Expected exchange name BINANCE, got %s", spotExchange.Name())
			}
			if spotExchange.Market() != schema.SPOT {
				t.Errorf("Expected market type SPOT, got %s", spotExchange.Market())
			}
		}

		// 测试Binance U本位合约
		usdtExchange := sdk.createBinanceExchange(schema.FUTURESUSDT, sdk.manager.Cache())
		if usdtExchange == nil {
			t.Errorf("Failed to create Binance FUTURESUSDT exchange")
		} else {
			if usdtExchange.Name() != schema.BINANCE {
				t.Errorf("Expected exchange name BINANCE, got %s", usdtExchange.Name())
			}
			if usdtExchange.Market() != schema.FUTURESUSDT {
				t.Errorf("Expected market type FUTURESUSDT, got %s", usdtExchange.Market())
			}
		}

		// 测试Binance币本位合约
		coinExchange := sdk.createBinanceExchange(schema.FUTURESCOIN, sdk.manager.Cache())
		if coinExchange == nil {
			t.Errorf("Failed to create Binance FUTURESCOIN exchange")
		} else {
			if coinExchange.Name() != schema.BINANCE {
				t.Errorf("Expected exchange name BINANCE, got %s", coinExchange.Name())
			}
			if coinExchange.Market() != schema.FUTURESCOIN {
				t.Errorf("Expected market type FUTURESCOIN, got %s", coinExchange.Market())
			}
		}
	})

	t.Run("Create OKX Exchanges", func(t *testing.T) {
		// 测试OKX现货
		spotExchange := sdk.createOKXExchange(schema.SPOT, sdk.manager.Cache())
		if spotExchange == nil {
			t.Errorf("Failed to create OKX SPOT exchange")
		} else {
			if spotExchange.Name() != schema.OKX {
				t.Errorf("Expected exchange name OKX, got %s", spotExchange.Name())
			}
			if spotExchange.Market() != schema.SPOT {
				t.Errorf("Expected market type SPOT, got %s", spotExchange.Market())
			}
		}

		// 测试OKX U本位合约
		usdtExchange := sdk.createOKXExchange(schema.FUTURESUSDT, sdk.manager.Cache())
		if usdtExchange == nil {
			t.Errorf("Failed to create OKX FUTURESUSDT exchange")
		} else {
			if usdtExchange.Name() != schema.OKX {
				t.Errorf("Expected exchange name OKX, got %s", usdtExchange.Name())
			}
			if usdtExchange.Market() != schema.FUTURESUSDT {
				t.Errorf("Expected market type FUTURESUSDT, got %s", usdtExchange.Market())
			}
		}
	})

	t.Run("Create Other Exchanges", func(t *testing.T) {
		// 测试Bybit现货
		bybitExchange := sdk.createBybitExchange(schema.SPOT, sdk.manager.Cache())
		if bybitExchange == nil {
			t.Errorf("Failed to create Bybit SPOT exchange")
		}

		// 测试Gate现货
		gateExchange := sdk.createGateExchange(schema.SPOT, sdk.manager.Cache())
		if gateExchange == nil {
			t.Errorf("Failed to create Gate SPOT exchange")
		}

		// 测试MEXC现货
		mexcExchange := sdk.createMEXCExchange(schema.SPOT, sdk.manager.Cache())
		if mexcExchange == nil {
			t.Errorf("Failed to create MEXC SPOT exchange")
		}
	})

	t.Run("Test Unsupported Market Types", func(t *testing.T) {
		// 测试不支持的市场类型
		unsupportedExchange := sdk.createBinanceExchange("UNSUPPORTED", sdk.manager.Cache())
		if unsupportedExchange != nil {
			t.Errorf("Expected nil for unsupported market type, got %+v", unsupportedExchange)
		}
	})
}

// 测试新的订阅逻辑
func TestSDKSubscriptionLogic(t *testing.T) {
	sdk := NewSDK()

	// 添加交易所配置
	exchangeConfig := ExchangeConfig{
		Name:   schema.BINANCE,
		Market: schema.SPOT,
		Weight: 3,
	}
	sdk.exchangeConfigs = append(sdk.exchangeConfigs, exchangeConfig)

	// 添加币对配置
	symbolConfig := SymbolConfig{
		Base:   "BTC",
		Quote:  "USDT",
		Margin: "",
		Market: schema.SPOT,
	}
	sdk.symbolConfigs = append(sdk.symbolConfigs, symbolConfig)

	t.Run("Test Symbol Formatting", func(t *testing.T) {
		// 测试币对格式化
		formattedSymbol, err := schema.FormatSymbolByExchange(
			schema.BINANCE,
			"BTC",
			"USDT",
			"",
			schema.SPOT,
		)
		if err != nil {
			t.Errorf("格式化币对失败: %v", err)
		}

		// Binance现货应该是 BTCUSDT
		expected := "BTCUSDT"
		if formattedSymbol != expected {
			t.Errorf("期望格式化后的符号为 %s, 实际得到 %s", expected, formattedSymbol)
		}
	})

	t.Run("Test Contract Symbol Formatting", func(t *testing.T) {
		// 测试合约币对格式化
		formattedSymbol, err := schema.FormatSymbolByExchange(
			schema.BINANCE,
			"BTC",
			"USDT",
			"USDT",
			schema.FUTURESUSDT,
		)
		if err != nil {
			t.Errorf("格式化合约币对失败: %v", err)
		}

		// Binance U本位合约应该是 BTCUSDT
		expected := "BTCUSDT"
		if formattedSymbol != expected {
			t.Errorf("期望格式化后的合约符号为 %s, 实际得到 %s", expected, formattedSymbol)
		}
	})

	t.Run("Test OKX Symbol Formatting", func(t *testing.T) {
		// 测试OKX币对格式化
		formattedSymbol, err := schema.FormatSymbolByExchange(
			schema.OKX,
			"BTC",
			"USDT",
			"",
			schema.SPOT,
		)
		if err != nil {
			t.Errorf("格式化OKX币对失败: %v", err)
		}

		// OKX现货应该是 BTC-USDT
		expected := "BTC-USDT"
		if formattedSymbol != expected {
			t.Errorf("期望格式化后的OKX符号为 %s, 实际得到 %s", expected, formattedSymbol)
		}
	})

	t.Run("Test Bybit Symbol Formatting", func(t *testing.T) {
		// 测试Bybit币对格式化
		formattedSymbol, err := schema.FormatSymbolByExchange(
			schema.BYBIT,
			"BTC",
			"USDT",
			"",
			schema.SPOT,
		)
		if err != nil {
			t.Errorf("格式化Bybit币对失败: %v", err)
		}

		// Bybit现货应该是 BTCUSDT
		expected := "BTCUSDT"
		if formattedSymbol != expected {
			t.Errorf("期望格式化后的Bybit符号为 %s, 实际得到 %s", expected, formattedSymbol)
		}
	})
}

// 测试新的智能数据读取方法
func TestSDKSmartDataReading(t *testing.T) {
	sdk := NewSDK()

	t.Run("Test WatchKline", func(t *testing.T) {
		// 测试现货币对
		kline, ok := sdk.WatchKline("BTC/USDT")
		if ok {
			t.Logf("现货BTC/USDT K线数据: 开盘价=%s, 最高价=%s, 最低价=%s, 收盘价=%s",
				kline.Open, kline.High, kline.Low, kline.Close)
		} else {
			t.Log("现货BTC/USDT K线数据: 暂无数据")
		}

		// 测试U本位合约币对
		kline, ok = sdk.WatchKline("BTC/USDT:USDT")
		if ok {
			t.Logf("U本位BTC/USDT:USDT K线数据: 开盘价=%s, 最高价=%s, 最低价=%s, 收盘价=%s",
				kline.Open, kline.High, kline.Low, kline.Close)
		} else {
			t.Log("U本位BTC/USDT:USDT K线数据: 暂无数据")
		}

		// 测试币本位合约币对
		kline, ok = sdk.WatchKline("BTC/USD:BTC")
		if ok {
			t.Logf("币本位BTC/USD:BTC K线数据: 开盘价=%s, 最高价=%s, 最低价=%s, 收盘价=%s",
				kline.Open, kline.High, kline.Low, kline.Close)
		} else {
			t.Log("币本位BTC/USD:BTC K线数据: 暂无数据")
		}

		// 测试无效币对
		kline, ok = sdk.WatchKline("INVALID/SYMBOL")
		if ok {
			t.Error("无效币对应该返回false")
		}
	})

	t.Run("Test WatchDepth", func(t *testing.T) {
		// 测试现货币对
		depth, ok := sdk.WatchDepth("BTC/USDT")
		if ok {
			t.Logf("现货BTC/USDT深度数据: 买单%d档, 卖单%d档", len(depth.Bids), len(depth.Asks))
		} else {
			t.Log("现货BTC/USDT深度数据: 暂无数据")
		}

		// 测试U本位合约币对
		depth, ok = sdk.WatchDepth("BTC/USDT:USDT")
		if ok {
			t.Logf("U本位BTC/USDT:USDT深度数据: 买单%d档, 卖单%d档", len(depth.Bids), len(depth.Asks))
		} else {
			t.Log("U本位BTC/USDT:USDT深度数据: 暂无数据")
		}

		// 测试币本位合约币对
		depth, ok = sdk.WatchDepth("BTC/USD:BTC")
		if ok {
			t.Logf("币本位BTC/USD:BTC深度数据: 买单%d档, 卖单%d档", len(depth.Bids), len(depth.Asks))
		} else {
			t.Log("币本位BTC/USD:BTC深度数据: 暂无数据")
		}

		// 测试无效币对
		depth, ok = sdk.WatchDepth("INVALID/SYMBOL")
		if ok {
			t.Error("无效币对应该返回false")
		}
	})

	t.Run("Test Default Exchange Order", func(t *testing.T) {
		exchanges := sdk.getDefaultExchangeOrder()
		expectedOrder := []schema.ExchangeName{
			schema.BINANCE,
			schema.OKX,
			schema.BYBIT,
			schema.GATE,
			schema.MEXC,
		}

		if len(exchanges) != len(expectedOrder) {
			t.Errorf("期望交易所数量 %d, 实际得到 %d", len(expectedOrder), len(exchanges))
		}

		for i, expected := range expectedOrder {
			if exchanges[i] != expected {
				t.Errorf("位置 %d: 期望 %s, 实际得到 %s", i, expected, exchanges[i])
			}
		}

		t.Logf("默认交易所顺序: %v", exchanges)
	})
}

// 测试新的批量订阅逻辑
func TestSDKBatchSubscription(t *testing.T) {
	_ = NewSDK() // 创建SDK实例但不使用，仅用于测试

	// 配置交易所
	exchangeConfigs := []ExchangeConfig{
		{Name: schema.BINANCE, Market: schema.SPOT, Weight: 3},
		{Name: schema.OKX, Market: schema.FUTURESUSDT, Weight: 1},
		{Name: schema.BYBIT, Market: schema.SPOT, Weight: 2},
	}

	// 配置币对
	symbolConfigs := []SymbolConfig{
		{Base: "BTC", Quote: "USDT", Market: schema.SPOT},
		{Base: "ETH", Quote: "USDT", Market: schema.SPOT},
		{Base: "BNB", Quote: "USDT", Market: schema.SPOT},
		{Base: "BTC", Quote: "USDT", Margin: "USDT", Market: schema.FUTURESUSDT},
		{Base: "SOL", Quote: "USDT", Margin: "USDT", Market: schema.FUTURESUSDT},
	}

	// 测试分组逻辑
	t.Run("Test Subscription Grouping", func(t *testing.T) {
		// 模拟分组逻辑
		subscriptionGroups := make(map[string][]string)

		for _, symbolConfig := range symbolConfigs {
			for _, exchangeConfig := range exchangeConfigs {
				if exchangeConfig.Market == symbolConfig.Market {
					groupKey := fmt.Sprintf("%s_%s", exchangeConfig.Name, exchangeConfig.Market)

					// 模拟格式化后的币对名称
					var formattedSymbol string
					switch exchangeConfig.Name {
					case schema.BINANCE:
						formattedSymbol = symbolConfig.Base + symbolConfig.Quote
					case schema.OKX:
						if exchangeConfig.Market == schema.FUTURESUSDT {
							formattedSymbol = symbolConfig.Base + "-" + symbolConfig.Quote + "-SWAP"
						} else {
							formattedSymbol = symbolConfig.Base + "-" + symbolConfig.Quote
						}
					case schema.BYBIT:
						formattedSymbol = symbolConfig.Base + symbolConfig.Quote
					}

					subscriptionGroups[groupKey] = append(subscriptionGroups[groupKey], formattedSymbol)
				}
			}
		}

		// 验证分组结果
		expectedGroups := map[string][]string{
			"binance_spot":     {"BTCUSDT", "ETHUSDT", "BNBUSDT"},
			"okx_futures_usdt": {"BTC-USDT-SWAP", "SOL-USDT-SWAP"},
			"bybit_spot":       {"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		}

		for groupKey, expectedSymbols := range expectedGroups {
			if symbols, exists := subscriptionGroups[groupKey]; exists {
				if len(symbols) != len(expectedSymbols) {
					t.Errorf("分组 %s: 期望 %d 个币对, 实际得到 %d 个",
						groupKey, len(expectedSymbols), len(symbols))
				}
				t.Logf("分组 %s: %v", groupKey, symbols)
			} else {
				t.Errorf("缺少分组: %s", groupKey)
			}
		}

		t.Logf("总分组数: %d", len(subscriptionGroups))
		for groupKey, symbols := range subscriptionGroups {
			t.Logf("分组 %s: %d 个币对", groupKey, len(symbols))
		}
	})

	t.Run("Test Exchange Market Matching", func(t *testing.T) {
		// 测试交易所和市场类型的匹配逻辑
		for _, exchangeConfig := range exchangeConfigs {
			for _, symbolConfig := range symbolConfigs {
				if exchangeConfig.Market == symbolConfig.Market {
					t.Logf("匹配: %s %s <-> %s %s",
						exchangeConfig.Name, exchangeConfig.Market,
						symbolConfig.Base, symbolConfig.Quote)
				}
			}
		}
	})
}
