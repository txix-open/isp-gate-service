package helpers_test

import (
	"isp-gate-service/helpers"
	"testing"
)

// nolint:gochecknoglobals,lll
var (
	rawUnicode = []byte(`{"fiodr":{"lastName":"\\u041F\\u0435\\u0442\\u0440\\u043E\\u0432","firstName":"\\u0418\\u0432\\u0430\\u043D","middleName":"\\u0421\\u0435\\u0440\\u0433\\u0435\\u0435\\u0432\\u0438\\u0447","birthDay":"1950-08-18"},"objectType":"addresses.addr_registration","params":{"unom":"63988","flat":"78"}}`)
	rawAscii   = []byte(`{"fiodr":{"lastName":"Петров","firstName":"Иван","middleName":"Сергеевич","birthDay":"1950-08-18"},"objectType":"addresses.addr_registration","params":{"unom":"63988","flat":"78"}}`)
)

func Benchmark_UnescapeUnicodeJson_Unicode(b *testing.B) {
	for b.Loop() {
		_ = helpers.UnescapeUnicodeJson(rawUnicode)
	}
}

func Benchmark_UnescapeUnicodeJson_Anscii(b *testing.B) {
	for b.Loop() {
		_ = helpers.UnescapeUnicodeJson(rawAscii)
	}
}

func Benchmark_UnescapeUnicode_Unicode(b *testing.B) {
	for b.Loop() {
		_ = helpers.UnescapeUnicode(rawUnicode)
	}
}

func Benchmark_UnescapeUnicode_Anscii(b *testing.B) {
	for b.Loop() {
		_ = helpers.UnescapeUnicode(rawAscii)
	}
}
