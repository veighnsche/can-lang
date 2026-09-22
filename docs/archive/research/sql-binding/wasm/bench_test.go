package main

import (
	"testing"

	pgquery "github.com/wasilibs/go-pgquery"
)

const benchSelect = "SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2"

func BenchmarkWasmParseSelect(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pgquery.Parse(benchSelect); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWasmScanSelect(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pgquery.Scan(benchSelect); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWasmParseTiny(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pgquery.Parse("SELECT 1"); err != nil {
			b.Fatal(err)
		}
	}
}
