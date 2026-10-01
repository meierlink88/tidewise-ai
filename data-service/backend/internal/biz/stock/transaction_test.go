package stock

import (
	"strings"
	"testing"
)

func TestDailyQuoteDecimalAndPlaceholder(t *testing.T) {
	input := `{"meta":{"period":"day","universe":1,"record_count":1},"stocks":[{"code":"601091","exchange":"SH","name":"样本","kline":[{"date":"2026-09-16","open":0,"high":0,"low":0,"close":4.39,"volume":0,"turnover":0,"change_pct":373.8041002278}]}]}`
	batch, err := DecodeDailyQuotes(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	q := batch.Quotes[0].Quote
	if q.Status != QuotePlaceholder || q.Open != nil || q.High != nil || q.Low != nil || *q.ChangePct != "373.8041002278" {
		t.Fatal(q)
	}
	for _, bad := range []string{
		strings.Replace(input, `"high":0`, `"high":null`, 1),
		strings.Replace(input, `"volume":0`, `"volume":1`, 1),
		strings.Replace(input, `"change_pct":373.8041002278`, `"change_pct":0.00000000001`, 1),
		strings.Replace(input, `"close":4.39`, `"close":-1`, 1),
		strings.Replace(input, `"period":"day"`, `"period":"week"`, 1),
	} {
		if _, err = DecodeDailyQuotes(strings.NewReader(bad)); err == nil {
			t.Fatal("invalid batch accepted", bad)
		}
	}
}

func TestDailyQuoteOHLCNullRatesAndDuplicateDates(t *testing.T) {
	input := `{"meta":{"period":"day","universe":1,"record_count":1},"stocks":[{"code":"000001","exchange":"SZ","name":"样本","kline":[{"date":"2026-09-30","open":10,"high":11,"low":9,"close":10.25,"volume":123.25,"turnover":null,"change_pct":-2.18}]}]}`
	batch, err := DecodeDailyQuotes(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if batch.Quotes[0].Quote.Turnover != nil || *batch.Quotes[0].Quote.ChangePct != "-2.18" {
		t.Fatal(batch)
	}
	for _, bad := range []string{
		strings.Replace(input, `"high":11`, `"high":9`, 1),
		strings.Replace(input, `"low":9`, `"low":10.1`, 1),
		strings.Replace(input, `"close":10.25`, `"close":11.25`, 1),
		strings.Replace(input, `"volume":123.25`, `"volume":-1`, 1),
		strings.Replace(input, `"volume":123.25`, `"volume":"123.25"`, 1),
		strings.Replace(input, `"date":"2026-09-30"`, `"date":"2026-02-30"`, 1),
		strings.Replace(input, `"turnover":null,`, ``, 1),
		strings.Replace(input, `"record_count":1`, `"record_count":2`, 1),
		strings.Replace(input, `"record_count":1`, `"record_count":1,"stats":{"K线条数合计":2,"有K线股票数":1,"无K线股票数":0}`, 1),
		input + `{}`,
	} {
		if _, err = DecodeDailyQuotes(strings.NewReader(bad)); err == nil {
			t.Fatal("invalid input accepted", bad)
		}
	}
	duplicate := strings.Replace(input, `"kline":[`, `"kline":[{"date":"2026-09-30","open":10,"high":11,"low":9,"close":10.25,"volume":123.25,"turnover":null,"change_pct":-2.18},`, 1)
	if _, err = DecodeDailyQuotes(strings.NewReader(duplicate)); err == nil {
		t.Fatal("duplicate accepted")
	}
}
