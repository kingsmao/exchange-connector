package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exchange-connector/pkg/schema"
	"exchange-connector/pkg/sdk"
)

func main() {
	fmt.Println("=== Exchange Connector 快速开始 ===")

	// 1. 创建SDK
	sdkInstance := sdk.NewSDK()

	// 2. 配置交易所（权重）
	fmt.Println("配置交易所...")
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
	fmt.Println("配置币对...")

	// 方式1：批量添加币对（自动识别市场类型）
	allSymbols := []string{
		"BTC/USDT",      // 现货
		"ETH/USDT",      // 现货
		"BNB/USDT",      // 现货
		"BTC/USDT:USDT", // U本位合约
		"ETH/USD:ETH",   // 币本位合约
	}

	// 4. 使用便捷函数：添加币对并自动订阅WebSocket（一步完成）
	fmt.Println("添加币对并自动订阅数据...")
	ctx := context.Background()

	// 方式A：批量添加币对并订阅
	if err := sdkInstance.AddSymbolsAndSubscribe(ctx, allSymbols); err != nil {
		panic(err)
	}

	// 5. 启动数据监控循环
	fmt.Println("启动数据监控循环，每3秒打印一次...")
	fmt.Println("按 Ctrl+C 退出程序")
	fmt.Println()

	// 创建定时器，每3秒执行一次
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// 创建退出信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 循环监控数据
	for {
		select {
		case <-ticker.C:
			// 每3秒执行一次数据读取和打印
			fmt.Printf("\n=== %s ===\n", time.Now().Format("2006-01-02 15:04:05"))

			// 读取K线数据
			if kline, ok := sdkInstance.WatchKline("BTC/USDT"); ok {
				fmt.Printf("现货BTC/USDT K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s, 成交量=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close, kline.Volume)
			} else {
				fmt.Println("现货BTC/USDT K线: 暂无数据")
			}

			if kline, ok := sdkInstance.WatchKline("BTC/USDT:USDT"); ok {
				fmt.Printf("U本位BTC/USDT:USDT K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s, 成交量=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close, kline.Volume)
			} else {
				fmt.Println("U本位BTC/USDT:USDT K线: 暂无数据")
			}

			if kline, ok := sdkInstance.WatchKline("ETH/USD:ETH"); ok {
				fmt.Printf("币本位ETH/USD:ETH K线: 开盘=%s, 最高=%s, 最低=%s, 收盘=%s, 成交量=%s\n",
					kline.Open, kline.High, kline.Low, kline.Close, kline.Volume)
			} else {
				fmt.Println("币本位ETH/USD:ETH K线: 暂无数据")
			}

			// 读取深度数据
			if depth, ok := sdkInstance.WatchDepth("BTC/USDT"); ok {
				fmt.Printf("现货BTC/USDT深度: 买单%d档, 卖单%d档, 买一=%s, 卖一=%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Asks[0].Price)
			} else {
				fmt.Println("现货BTC/USDT深度: 暂无数据")
			}

			if depth, ok := sdkInstance.WatchDepth("BTC/USDT:USDT"); ok {
				fmt.Printf("U本位BTC/USDT:USDT深度: 买单%d档, 卖单%d档, 买一=%s, 卖一=%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Asks[0].Price)
			} else {
				fmt.Println("U本位BTC/USDT:USDT深度: 暂无数据")
			}

			if depth, ok := sdkInstance.WatchDepth("ETH/USD:ETH"); ok {
				fmt.Printf("币本位ETH/USD:ETH深度: 买单%d档, 卖单%d档, 买一=%s, 卖一=%s\n",
					len(depth.Bids), len(depth.Asks), depth.Bids[0].Price, depth.Asks[0].Price)
			} else {
				fmt.Println("币本位ETH/USD:ETH深度: 暂无数据")
			}

			fmt.Println("---")

		case <-quit:
			// 收到退出信号
			fmt.Println("\n收到退出信号，正在关闭...")
			return
		}
	}

}
