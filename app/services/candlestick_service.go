package services

import (
	"fmt"
	"time"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

// GetCandlestick mengembalikan bar OHLC harian untuk grafik candlestick.
// Sumbernya Yahoo Finance yang sama dengan riwayat harga, tetapi blok quote
// dibaca seluruhnya (open/high/low/close), bukan hanya close.
func (s *MarketService) GetCandlestick(asset string, days int) ([]map[string]interface{}, string, error) {
	days = clampHistoryDays(days)
	if days > 365 {
		days = 365
	}

	symbol, ok := historyYahooSymbol(asset)
	if !ok {
		return nil, "", fmt.Errorf("aset candlestick tidak dikenal: %q", asset)
	}

	span := historySpan(days)
	cacheKey := fmt.Sprintf("market_candlestick_full_%s_%s", asset, span)
	fresh := !facades.Cache().Has(cacheKey)

	cached, err := facades.Cache().Remember(cacheKey, historyCacheTTL, func() (interface{}, error) {
		points, fetchErr := fetchYahooCandles(symbol, span)
		if fetchErr != nil {
			return nil, fetchErr
		}

		return historySeries{Source: "yahoo_finance", Points: points}, nil
	})
	if err != nil {
		if stored, ok := cachedHistorySeries(cacheKey); ok {
			facades.Log().Warningf("candlestick %s gagal diambil (%v), memakai cache terakhir dari database", asset, err)
			return lastNDays(stored.Points, days), stored.Source, nil
		}

		return nil, "", err
	}

	series, ok := cached.(historySeries)
	if !ok {
		return nil, "", fmt.Errorf("unexpected candlestick cache payload of type %T", cached)
	}

	if fresh {
		persistMarketCache(cacheKey, series)
	}

	return lastNDays(series.Points, days), series.Source, nil
}

func fetchYahooCandles(symbol string, span string) ([]map[string]interface{}, error) {
	payload, err := fetchJsonWithRetry(fmt.Sprintf(yahooChartURL, symbol, span))
	if err != nil {
		return nil, err
	}

	chart, ok := toMap(payload["chart"])
	if !ok {
		return nil, fmt.Errorf("payload Yahoo Finance untuk %s tidak punya bagian chart", symbol)
	}

	results, ok := toSlice(chart["result"])
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan hasil untuk %s", symbol)
	}

	first, ok := toMap(results[0])
	if !ok {
		return nil, fmt.Errorf("hasil Yahoo Finance untuk %s tidak valid", symbol)
	}

	timestamps, ok := toSlice(first["timestamp"])
	if !ok || len(timestamps) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan timestamp untuk %s", symbol)
	}

	opens, highs, lows, closes, err := yahooOhlc(first)
	if err != nil {
		return nil, fmt.Errorf("%w (symbol %s)", err, symbol)
	}

	location := yahooLocation(first)
	scale := yahooPriceScale(yahooMetaCurrency(first))

	series := make([]map[string]interface{}, 0, len(timestamps))
	for index, entry := range timestamps {
		if index >= len(opens) || index >= len(highs) || index >= len(lows) || index >= len(closes) {
			break
		}

		seconds := int64(toFloat64(entry))
		open := toFloat64(opens[index]) * scale
		high := toFloat64(highs[index]) * scale
		low := toFloat64(lows[index]) * scale
		closePrice := toFloat64(closes[index]) * scale

		if seconds <= 0 || open <= 0 || high <= 0 || low <= 0 || closePrice <= 0 {
			continue
		}

		if high < open {
			high = open
		}
		if high < closePrice {
			high = closePrice
		}
		if low > open {
			low = open
		}
		if low > closePrice {
			low = closePrice
		}

		series = append(series, map[string]interface{}{
			"date":  time.Unix(seconds, 0).In(location).Format(historyDateLayout),
			"ts":    seconds * 1000,
			"open":  open,
			"high":  high,
			"low":   low,
			"close": closePrice,
		})
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak punya bar OHLC untuk %s", symbol)
	}

	return series, nil
}

func yahooOhlc(result map[string]any) ([]any, []any, []any, []any, error) {
	indicators, ok := toMap(result["indicators"])
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("hasil Yahoo Finance tidak punya bagian indicators")
	}

	quotes, ok := toSlice(indicators["quote"])
	if !ok || len(quotes) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("hasil Yahoo Finance tidak punya array quote")
	}

	quote, ok := toMap(quotes[0])
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("array quote Yahoo Finance tidak valid")
	}

	opens, okOpen := toSlice(quote["open"])
	highs, okHigh := toSlice(quote["high"])
	lows, okLow := toSlice(quote["low"])
	closes, okClose := toSlice(quote["close"])
	if !okOpen || !okHigh || !okLow || !okClose || len(closes) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("hasil Yahoo Finance tidak punya bar OHLC lengkap")
	}

	return opens, highs, lows, closes, nil
}
