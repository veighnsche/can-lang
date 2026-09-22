package main

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

const benchSelect = "SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2"

func BenchmarkCgoParseSelect(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pg_query.Parse(benchSelect); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCgoScanSelect(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pg_query.Scan(benchSelect); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCgoParseTiny(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pg_query.Parse("SELECT 1"); err != nil {
			b.Fatal(err)
		}
	}
}
