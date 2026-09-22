// Command wasm runs the shared SQL corpus through the WASM/wazero binding,
// github.com/wasilibs/go-pgquery. It must reproduce the reference outputs
// byte for byte; the embedded module holds the same PostgreSQL 17.7 parser,
// so any difference is a real discrepancy. Modes match ../cgo/main.go. The
// two mains intentionally duplicate logic so each imports one candidate.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	pgquery "github.com/wasilibs/go-pgquery"
	"google.golang.org/protobuf/encoding/protojson"
)

type Case struct {
	ID    string `json:"id"`
	SQL   string `json:"sql"`
	Notes string `json:"notes"`
}

type Corpus struct {
	Cases []Case `json:"cases"`
}

type Failure struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Cursorpos int    `json:"cursorpos"`
}

type Stmt struct {
	Location int    `json:"location"`
	Length   int    `json:"length"`
	Kind     string `json:"kind"`
}

type Token struct {
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Text    string `json:"text"`
	Name    string `json:"token"`
	Keyword string `json:"keyword"`
}

type Record struct {
	ID       string   `json:"id"`
	SHA256   string   `json:"sha256"`
	Version  int32    `json:"version"`
	ParseErr *Failure `json:"parse_error"`
	Stmts    []Stmt   `json:"stmts"`
	ScanErr  *Failure `json:"scan_error"`
	Tokens   []Token  `json:"tokens"`
	RawParse string   `json:"raw_parse"`
	RawScan  string   `json:"raw_scan"`
}

func failureOf(err error) *Failure {
	if err == nil {
		return nil
	}
	cursor := -1
	value := reflect.ValueOf(err)
	if value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	if value.Kind() == reflect.Struct {
		if field := value.FieldByName("Cursorpos"); field.IsValid() && field.Kind() == reflect.Int {
			cursor = int(field.Int())
		}
	}
	return &Failure{Type: fmt.Sprintf("%T", err), Message: err.Error(), Cursorpos: cursor}
}

func runOne(sql string) Record {
	record := Record{}
	sum := sha256.Sum256([]byte(sql))
	record.SHA256 = hex.EncodeToString(sum[:])
	tree, parseErr := pgquery.Parse(sql)
	record.ParseErr = failureOf(parseErr)
	if parseErr == nil {
		record.Version = tree.Version
		for _, stmt := range tree.Stmts {
			kind := "none"
			if stmt.Stmt != nil {
				kind = fmt.Sprintf("%T", stmt.Stmt.Node)
			}
			record.Stmts = append(record.Stmts, Stmt{Location: int(stmt.StmtLocation), Length: int(stmt.StmtLen), Kind: kind})
		}
		raw, err := protojson.Marshal(tree)
		if err != nil {
			panic(err)
		}
		record.RawParse = string(raw)
	}
	scan, scanErr := pgquery.Scan(sql)
	record.ScanErr = failureOf(scanErr)
	if scanErr == nil {
		if record.Version == 0 {
			record.Version = scan.Version
		}
		for _, token := range scan.Tokens {
			text := ""
			if token.Start >= 0 && token.End >= token.Start && int(token.End) <= len(sql) {
				text = sql[token.Start:token.End]
			}
			record.Tokens = append(record.Tokens, Token{Start: int(token.Start), End: int(token.End), Text: text, Name: token.Token.String(), Keyword: token.KeywordKind.String()})
		}
		raw, err := protojson.Marshal(scan)
		if err != nil {
			panic(err)
		}
		record.RawScan = string(raw)
	}
	return record
}

func main() {
	corpusPath := flag.String("corpus", "", "shared corpus JSON")
	outPath := flag.String("out", "", "results JSON output")
	rawDir := flag.String("rawdir", "", "directory for per-case raw outputs")
	once := flag.String("once", "", "parse and scan one SQL string, print cold timing")
	batch := flag.Int("batch", 0, "repeat the corpus N times, print memory stats")
	flag.Parse()

	if *once != "" {
		start := time.Now()
		record := runOne(*once)
		elapsed := time.Since(start)
		study := map[string]any{"elapsed_ns": elapsed.Nanoseconds(), "version": record.Version, "stmts": len(record.Stmts), "tokens": len(record.Tokens), "parse_error": record.ParseErr != nil, "scan_error": record.ScanErr != nil}
		raw, _ := json.Marshal(study)
		fmt.Println(string(raw))
		return
	}

	if *corpusPath == "" || *outPath == "" {
		panic("corpus and out are required")
	}
	data, err := os.ReadFile(*corpusPath)
	if err != nil {
		panic(err)
	}
	var corpus Corpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		panic(err)
	}
	if *batch > 0 {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := 0; i < *batch; i++ {
			for _, c := range corpus.Cases {
				runOne(c.SQL)
			}
		}
		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)
		study := map[string]any{"iterations": *batch * len(corpus.Cases), "elapsed_ns": elapsed.Nanoseconds(), "heap_alloc": after.HeapAlloc, "total_alloc": after.TotalAlloc - before.TotalAlloc, "num_gc": after.NumGC - before.NumGC}
		raw, _ := json.Marshal(study)
		fmt.Println(string(raw))
		return
	}

	records := make([]Record, 0, len(corpus.Cases))
	for _, c := range corpus.Cases {
		if strings.TrimSpace(c.ID) == "" {
			panic("corpus case without id")
		}
		record := runOne(c.SQL)
		record.ID = c.ID
		records = append(records, record)
		if *rawDir != "" {
			if err := os.MkdirAll(*rawDir, 0755); err != nil {
				panic(err)
			}
			safe := c.ID
			if err := os.WriteFile(filepath.Join(*rawDir, safe+".parse.json"), []byte(record.RawParse+"\n"), 0644); err != nil {
				panic(err)
			}
			if err := os.WriteFile(filepath.Join(*rawDir, safe+".scan.json"), []byte(record.RawScan+"\n"), 0644); err != nil {
				panic(err)
			}
		}
	}
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*outPath, append(raw, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "wasm: %d cases, params per case: %s\n", len(records), summarize(records, corpus))
}

func summarize(records []Record, corpus Corpus) string {
	byID := map[string]string{}
	for _, c := range corpus.Cases {
		byID[c.ID] = c.SQL
	}
	parts := make([]string, 0, len(records))
	for _, record := range records {
		sql := byID[record.ID]
		var params []string
		for _, token := range record.Tokens {
			if token.Name == "PARAM" && strings.HasPrefix(token.Text, "$") {
				if _, err := strconv.Atoi(token.Text[1:]); err == nil && token.Text == sql[token.Start:token.End] {
					params = append(params, fmt.Sprintf("%s@%d", token.Text, token.Start))
				}
			}
		}
		parts = append(parts, fmt.Sprintf("%s=%d[%s]", record.ID, len(record.Stmts), strings.Join(params, ",")))
	}
	return strings.Join(parts, " ")
}
