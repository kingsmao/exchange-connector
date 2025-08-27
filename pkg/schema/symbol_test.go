package schema

import (
	"testing"
)

func TestNewSymbol(t *testing.T) {
	symbol := NewSymbol("BTCUSDT", "BTC", "USDT", "", BINANCE, SPOT)

	if symbol.Symbol != "BTCUSDT" {
		t.Errorf("Expected Symbol to be 'BTCUSDT', got '%s'", symbol.Symbol)
	}
	if symbol.Base != "BTC" {
		t.Errorf("Expected Base to be 'BTC', got '%s'", symbol.Base)
	}
	if symbol.Quote != "USDT" {
		t.Errorf("Expected Quote to be 'USDT', got '%s'", symbol.Quote)
	}
	if symbol.ExchangeName != BINANCE {
		t.Errorf("Expected ExchangeName to be BINANCE, got %s", symbol.ExchangeName)
	}
	if symbol.MarketType != SPOT {
		t.Errorf("Expected MarketType to be SPOT, got %s", symbol.MarketType)
	}
}

func TestNewSymbolFromString(t *testing.T) {
	symbol := NewSymbolFromString("BTCUSDT", "BTC", "USDT", "", "BINANCE", "SPOT")

	if symbol.Symbol != "BTCUSDT" {
		t.Errorf("Expected Symbol to be 'BTCUSDT', got '%s'", symbol.Symbol)
	}
	if symbol.Base != "BTC" {
		t.Errorf("Expected Base to be 'BTC', got '%s'", symbol.Base)
	}
	if symbol.Quote != "USDT" {
		t.Errorf("Expected Quote to be 'USDT', got '%s'", symbol.Quote)
	}
	if symbol.ExchangeName != BINANCE {
		t.Errorf("Expected ExchangeName to be BINANCE, got %s", symbol.ExchangeName)
	}
	if symbol.MarketType != SPOT {
		t.Errorf("Expected MarketType to be SPOT, got %s", symbol.MarketType)
	}
}

func TestSymbol_String(t *testing.T) {
	tests := []struct {
		name     string
		symbol   *Symbol
		expected string
	}{
		{
			name:     "Spot symbol",
			symbol:   NewSymbol("BTCUSDT", "BTC", "USDT", "", BINANCE, SPOT),
			expected: "BTC/USDT",
		},
		{
			name:     "Futures symbol with margin",
			symbol:   NewSymbol("BTCUSD_PERP", "BTC", "USD", "USD", BINANCE, FUTURESCOIN),
			expected: "BTC/USD:USD",
		},
		{
			name:     "U本位合约符号（USDT合约）",
			symbol:   NewSymbol("BTCUSDT", "BTC", "USDT", "USDT", BINANCE, FUTURESUSDT),
			expected: "BTC/USDT:USDT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.symbol.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSymbol_IsSpot(t *testing.T) {
	tests := []struct {
		name     string
		symbol   *Symbol
		expected bool
	}{
		{
			name:     "Spot market",
			symbol:   NewSymbol("", "BTC", "USDT", "", BINANCE, SPOT),
			expected: true,
		},
		{
			name:     "Futures market",
			symbol:   NewSymbol("", "BTC", "USDT", "USDT", BINANCE, FUTURESUSDT),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.symbol.IsSpot()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSymbol_IsFutures(t *testing.T) {
	tests := []struct {
		name     string
		symbol   *Symbol
		expected bool
	}{
		{
			name:     "Spot market",
			symbol:   NewSymbol("", "BTC", "USDT", "", BINANCE, SPOT),
			expected: false,
		},
		{
			name:     "U本位合约市场（USDT合约）",
			symbol:   NewSymbol("", "BTC", "USDT", "USDT", BINANCE, FUTURESUSDT),
			expected: true,
		},
		{
			name:     "币本位合约市场",
			symbol:   NewSymbol("", "BTC", "USD", "BTC", BINANCE, FUTURESCOIN),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.symbol.IsFutures()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParser_ParseSymbol(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *Symbol
		expectError bool
	}{
		{
			name:  "Spot symbol with slash",
			input: "BTC/USDT",
			expected: &Symbol{
				Symbol:     "BTC/USDT",
				Base:       "BTC",
				Quote:      "USDT",
				Margin:     "",
				MarketType: SPOT,
			},
			expectError: false,
		},
		{
			name:  "U本位合约 symbol with margin",
			input: "BTC/USDT:USDT",
			expected: &Symbol{
				Symbol:     "BTC/USDT:USDT",
				Base:       "BTC",
				Quote:      "USDT",
				Margin:     "USDT",
				MarketType: FUTURESUSDT,
			},
			expectError: false,
		},
		{
			name:  "币本位合约 symbol with margin",
			input: "BTC/USD:BTC",
			expected: &Symbol{
				Symbol:     "BTC/USD:BTC",
				Base:       "BTC",
				Quote:      "USD",
				Margin:     "BTC",
				MarketType: FUTURESCOIN,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSymbol(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Base != tt.expected.Base {
				t.Errorf("Expected Base '%s', got '%s'", tt.expected.Base, result.Base)
			}
			if result.Quote != tt.expected.Quote {
				t.Errorf("Expected Quote '%s', got '%s'", tt.expected.Quote, result.Quote)
			}
			if result.Margin != tt.expected.Margin {
				t.Errorf("Expected Margin '%s', got '%s'", tt.expected.Margin, result.Margin)
			}
			if result.MarketType != tt.expected.MarketType {
				t.Errorf("Expected MarketType '%s', got '%s'", tt.expected.MarketType, result.MarketType)
			}
		})
	}
}

func TestParser_FormatSymbol(t *testing.T) {
	tests := []struct {
		name         string
		symbol       *Symbol
		exchangeName ExchangeName
		expected     string
		expectError  bool
	}{
		{
			name:         "Binance spot",
			symbol:       NewSymbol("", "BTC", "USDT", "", BINANCE, SPOT),
			exchangeName: BINANCE,
			expected:     "BTCUSDT",
			expectError:  false,
		},
		{
			name:         "OKX spot",
			symbol:       NewSymbol("", "BTC", "USDT", "", OKX, SPOT),
			exchangeName: OKX,
			expected:     "BTC-USDT",
			expectError:  false,
		},
		{
			name:         "Binance U本位合约",
			symbol:       NewSymbol("", "BTC", "USDT", "USDT", BINANCE, FUTURESUSDT),
			exchangeName: BINANCE,
			expected:     "BTCUSDT",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatSymbol(tt.symbol, tt.exchangeName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParser_ReverseParseSymbol(t *testing.T) {
	tests := []struct {
		name         string
		base         string
		quote        string
		margin       string
		exchangeName string
		marketType   string
		expected     *Symbol
		expectError  bool
	}{
		{
			name:         "Binance spot",
			base:         "BTC",
			quote:        "USDT",
			margin:       "",
			exchangeName: "BINANCE",
			marketType:   "SPOT",
			expected: &Symbol{
				Symbol:       "BTCUSDT",
				Base:         "BTC",
				Quote:        "USDT",
				Margin:       "",
				ExchangeName: BINANCE,
				MarketType:   SPOT,
			},
			expectError: false,
		},
		{
			name:         "U本位合约自动设置保证金",
			base:         "BTC",
			quote:        "USDT",
			margin:       "", // 自动设置为 USDT
			exchangeName: "BINANCE",
			marketType:   "FUTURESUSDT",
			expected: &Symbol{
				Symbol:       "BTCUSDT",
				Base:         "BTC",
				Quote:        "USDT",
				Margin:       "USDT",
				ExchangeName: BINANCE,
				MarketType:   FUTURESUSDT,
			},
			expectError: false,
		},
		{
			name:         "币本位合约自动设置保证金",
			base:         "BTC",
			quote:        "USD",
			margin:       "", // 自动设置为 BTC
			exchangeName: "BINANCE",
			marketType:   "FUTURESCOIN",
			expected: &Symbol{
				Symbol:       "BTCUSD",
				Base:         "BTC",
				Quote:        "USD",
				Margin:       "BTC",
				ExchangeName: BINANCE,
				MarketType:   FUTURESCOIN,
			},
			expectError: false,
		},
		{
			name:         "Case insensitive parsing",
			base:         "BTC",
			quote:        "USDT",
			margin:       "",
			exchangeName: "binance",
			marketType:   "spot",
			expected: &Symbol{
				Symbol:       "BTCUSDT",
				Base:         "BTC",
				Quote:        "USDT",
				Margin:       "",
				ExchangeName: BINANCE,
				MarketType:   SPOT,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReverseParseSymbol(tt.base, tt.quote, tt.margin, tt.exchangeName, tt.marketType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Symbol != tt.expected.Symbol {
				t.Errorf("Expected Symbol '%s', got '%s'", tt.expected.Symbol, result.Symbol)
			}
			if result.Base != tt.expected.Base {
				t.Errorf("Expected Base '%s', got '%s'", tt.expected.Base, result.Base)
			}
			if result.Quote != tt.expected.Quote {
				t.Errorf("Expected Quote '%s', got '%s'", tt.expected.Quote, result.Quote)
			}
			if result.Margin != tt.expected.Margin {
				t.Errorf("Expected Margin '%s', got '%s'", tt.expected.Margin, result.Margin)
			}
			if result.ExchangeName != tt.expected.ExchangeName {
				t.Errorf("Expected ExchangeName %s, got %s", tt.expected.ExchangeName, result.ExchangeName)
			}
			if result.MarketType != tt.expected.MarketType {
				t.Errorf("Expected MarketType %s, got %s", tt.expected.MarketType, result.MarketType)
			}
		})
	}
}

func TestNormalizeFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ExchangeName
	}{
		{"Uppercase", "BINANCE", BINANCE},
		{"Lowercase", "binance", BINANCE},
		{"Mixed case", "BiNaNcE", BINANCE},
		{"With spaces", " BINANCE ", BINANCE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeExchangeName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}

	marketTests := []struct {
		name     string
		input    string
		expected MarketType
	}{
		{"Uppercase", "SPOT", SPOT},
		{"Lowercase", "spot", SPOT},
		{"Mixed case", "SpOt", SPOT},
		{"With spaces", " SPOT ", SPOT},
	}

	for _, tt := range marketTests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeMarketType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMarginLogic(t *testing.T) {
	tests := []struct {
		name           string
		base           string
		quote          string
		margin         string
		exchangeName   string
		marketType     string
		expectedMargin string
		description    string
	}{
		{
			name:           "现货市场无保证金",
			base:           "BTC",
			quote:          "USDT",
			margin:         "",
			exchangeName:   "BINANCE",
			marketType:     "SPOT",
			expectedMargin: "",
			description:    "现货市场没有保证金币种",
		},
		{
			name:           "U本位合约保证金=quote",
			base:           "BTC",
			quote:          "USDT",
			margin:         "",
			exchangeName:   "BINANCE",
			marketType:     "FUTURESUSDT",
			expectedMargin: "USDT",
			description:    "U本位合约：保证金币种 = quote币种 (BTC/USDT:USDT)",
		},
		{
			name:           "币本位合约保证金=base",
			base:           "BTC",
			quote:          "USD",
			margin:         "",
			exchangeName:   "BINANCE",
			marketType:     "FUTURESCOIN",
			expectedMargin: "BTC",
			description:    "币本位合约：保证金币种 = base币种 (BTC/USD:BTC)",
		},
		{
			name:           "ETH币本位合约",
			base:           "ETH",
			quote:          "USD",
			margin:         "",
			exchangeName:   "BINANCE",
			marketType:     "FUTURESCOIN",
			expectedMargin: "ETH",
			description:    "ETH币本位合约：保证金币种 = base币种 (ETH/USD:ETH)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReverseParseSymbol(tt.base, tt.quote, tt.margin, tt.exchangeName, tt.marketType)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Margin != tt.expectedMargin {
				t.Errorf("Expected margin '%s', got '%s' - %s",
					tt.expectedMargin, result.Margin, tt.description)
			}

			// 验证保证金币种的逻辑
			switch tt.marketType {
			case "SPOT":
				if result.Margin != "" {
					t.Errorf("Spot market should have no margin, got '%s'", result.Margin)
				}
			case "FUTURESUSDT":
				if result.Margin != result.Quote {
					t.Errorf("U本位合约保证金币种应该等于quote币种: margin='%s', quote='%s'",
						result.Margin, result.Quote)
				}
			case "FUTURESCOIN":
				if result.Margin != result.Base {
					t.Errorf("币本位合约保证金币种应该等于base币种: margin='%s', base='%s'",
						result.Margin, result.Base)
				}
			}
		})
	}
}
