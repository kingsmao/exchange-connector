package schema

// Binance API Response Types

// BinanceTickerResponse represents Binance ticker API response
type BinanceTickerResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// BinanceFuturesTickerResponse represents Binance futures ticker API response
type BinanceFuturesTickerResponse struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Volume    string `json:"volume"`
	QuoteVol  string `json:"quoteVolume"`
	Timestamp int64  `json:"time"`
}

// BinanceKlineResponse represents Binance kline API response
// [0] Open time, [1] Open, [2] High, [3] Low, [4] Close, [5] Volume, [6] Close time, [7] Quote asset volume, [8] Number of trades
type BinanceKlineResponse [][]interface{}

// BinanceDepthResponse represents Binance depth API response
type BinanceDepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// OKX API Response Types

// OKXTickerResponse represents OKX ticker API response
type OKXTickerResponse struct {
	Data []struct {
		Last   string `json:"last"`
		Vol24h string `json:"vol24h"`
		InstID string `json:"instId"`
		Ts     string `json:"ts"`
	} `json:"data"`
}

// OKXKlineResponse represents OKX kline API response
type OKXKlineResponse struct {
	Data [][]string `json:"data"`
}

// OKXDepthResponse represents OKX depth API response
type OKXDepthResponse struct {
	Data []struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
		Ts   string     `json:"ts"`
	} `json:"data"`
}

// Bybit API Response Types

// BybitTickerResponse represents Bybit ticker API response
type BybitTickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
			Volume24h string `json:"turnover24h"`
			Time      int64  `json:"time"`
		} `json:"list"`
	} `json:"result"`
}

// BybitKlineResponse represents Bybit kline API response
type BybitKlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string     `json:"category"`
		Symbol   string     `json:"symbol"`
		List     [][]string `json:"list"`
	} `json:"result"`
}

// BybitDepthResponse represents Bybit depth API response
type BybitDepthResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string     `json:"category"`
		Symbol   string     `json:"symbol"`
		Bids     [][]string `json:"b"`
		Asks     [][]string `json:"a"`
		Ts       int64      `json:"ts"`
	} `json:"result"`
}

// Gate API Response Types

// GateTickerResponse represents Gate ticker API response
type GateTickerResponse struct {
	CurrencyPair string `json:"currency_pair"`
	Last         string `json:"last"`
	BaseVolume   string `json:"base_volume"`
	Time         int64  `json:"timestamp"`
}

// GateKlineResponse represents Gate kline API response
type GateKlineResponse [][]string

// GateDepthResponse represents Gate depth API response
type GateDepthResponse struct {
	CurrencyPair string     `json:"currency_pair"`
	Asks         [][]string `json:"asks"`
	Bids         [][]string `json:"bids"`
	Update       int64      `json:"update"`
}

// MEXC API Response Types

// MEXCTickerResponse represents MEXC ticker API response
type MEXCTickerResponse struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Volume    string `json:"volume"`
	Timestamp int64  `json:"timestamp"`
}

// MEXCKlineResponse represents MEXC kline API response
type MEXCKlineResponse [][]interface{}

// MEXCDepthResponse represents MEXC depth API response
type MEXCDepthResponse struct {
	Symbol string     `json:"symbol"`
	Bids   [][]string `json:"bids"`
	Asks   [][]string `json:"asks"`
	Ts     int64      `json:"Ts"`
}
