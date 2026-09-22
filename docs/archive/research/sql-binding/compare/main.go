// Command compare judges the two SQL binding harnesses. It never normalizes
// away a mismatch: every difference is reported with both raw values.
//
//	compare --pin ref.json --out pinned.json        derive pins from the reference run
//	compare --check pinned.json --candidate run.json  verify one run against pins
//	compare --verdict --ref ref.json --candidate run.json --out verdict.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

type Param struct {
	N     int `json:"n"`
	Start int `json:"start"`
	End   int `json:"end"`
}

type Pin struct {
	ID         string   `json:"id"`
	Version    int32    `json:"version"`
	Stmts      []Stmt   `json:"stmts"`
	Params     []Param  `json:"params"`
	ParseErr   *Failure `json:"parse_error"`
	ScanErr    *Failure `json:"scan_error"`
	TokenCount int      `json:"token_count"`
}

func paramsOf(record Record) []Param {
	var out []Param
	for _, token := range record.Tokens {
		if token.Name != "PARAM" || !strings.HasPrefix(token.Text, "$") {
			continue
		}
		n, err := strconv.Atoi(token.Text[1:])
		if err != nil {
			continue
		}
		out = append(out, Param{N: n, Start: token.Start, End: token.End})
	}
	return out
}

func loadRecords(path string) []Record {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		panic(err)
	}
	return records
}

func sameFailure(a, b *Failure) (bool, string) {
	if a == nil || b == nil {
		if a == b {
			return true, ""
		}
		return false, "one side errors"
	}
	if a.Message != b.Message || a.Cursorpos != b.Cursorpos {
		return false, fmt.Sprintf("message %q@%d vs %q@%d", a.Message, a.Cursorpos, b.Message, b.Cursorpos)
	}
	return true, ""
}

func pinMode(ref, out string) {
	records := loadRecords(ref)
	pins := make([]Pin, 0, len(records))
	for _, record := range records {
		params := paramsOf(record)
		if params == nil {
			params = []Param{}
		}
		stmts := record.Stmts
		if stmts == nil {
			stmts = []Stmt{}
		}
		pins = append(pins, Pin{ID: record.ID, Version: record.Version, Stmts: stmts, Params: params, ParseErr: record.ParseErr, ScanErr: record.ScanErr, TokenCount: len(record.Tokens)})
	}
	raw, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "pinned %d cases\n", len(pins))
}

func checkMode(pinnedPath, candidatePath string) {
	data, err := os.ReadFile(pinnedPath)
	if err != nil {
		panic(err)
	}
	var pins []Pin
	if err := json.Unmarshal(data, &pins); err != nil {
		panic(err)
	}
	records := loadRecords(candidatePath)
	byID := map[string]Record{}
	for _, record := range records {
		byID[record.ID] = record
	}
	failed := 0
	for _, pin := range pins {
		record, ok := byID[pin.ID]
		if !ok {
			fmt.Printf("FAIL %s: missing from candidate run\n", pin.ID)
			failed++
			continue
		}
		var diffs []string
		if record.Version != pin.Version {
			diffs = append(diffs, fmt.Sprintf("version %d vs %d", record.Version, pin.Version))
		}
		if ok, why := sameFailure(record.ParseErr, pin.ParseErr); !ok {
			diffs = append(diffs, "parse error: "+why)
		}
		if ok, why := sameFailure(record.ScanErr, pin.ScanErr); !ok {
			diffs = append(diffs, "scan error: "+why)
		}
		if len(record.Stmts) != len(pin.Stmts) {
			diffs = append(diffs, fmt.Sprintf("stmts %d vs %d", len(record.Stmts), len(pin.Stmts)))
		} else {
			for i := range record.Stmts {
				if record.Stmts[i] != pin.Stmts[i] {
					diffs = append(diffs, fmt.Sprintf("stmt %d %+v vs %+v", i, record.Stmts[i], pin.Stmts[i]))
				}
			}
		}
		params := paramsOf(record)
		if len(params) != len(pin.Params) {
			diffs = append(diffs, fmt.Sprintf("params %v vs %v", params, pin.Params))
		} else {
			for i := range params {
				if params[i] != pin.Params[i] {
					diffs = append(diffs, fmt.Sprintf("param %d %+v vs %+v", i, params[i], pin.Params[i]))
				}
			}
		}
		if len(record.Tokens) != pin.TokenCount {
			diffs = append(diffs, fmt.Sprintf("tokens %d vs %d", len(record.Tokens), pin.TokenCount))
		}
		if len(diffs) > 0 {
			fmt.Printf("FAIL %s: %s\n", pin.ID, strings.Join(diffs, "; "))
			failed++
		} else {
			fmt.Printf("ok %s\n", pin.ID)
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "all %d pins hold\n", len(pins))
}

func verdictMode(refPath, candidatePath, out string) {
	refs := loadRecords(refPath)
	cands := loadRecords(candidatePath)
	byID := map[string]Record{}
	for _, record := range cands {
		byID[record.ID] = record
	}
	type CaseVerdict struct {
		ID          string   `json:"id"`
		Match       bool     `json:"match"`
		Diffs       []string `json:"diffs"`
		TypeNote    string   `json:"type_note,omitempty"`
		RefSHA      string   `json:"ref_sha256"`
		CandSHA     string   `json:"cand_sha256"`
		ParseStmts  int      `json:"parse_stmts"`
		ScanTokens  int      `json:"scan_tokens"`
		RawParseEq  bool     `json:"raw_parse_equal"`
		RawScanEq   bool     `json:"raw_scan_equal"`
		ParamsMatch bool     `json:"params_match"`
	}
	verdicts := make([]CaseVerdict, 0, len(refs))
	for _, ref := range refs {
		verdict := CaseVerdict{ID: ref.ID, Match: true, RefSHA: ref.SHA256, ParseStmts: len(ref.Stmts), ScanTokens: len(ref.Tokens)}
		cand, ok := byID[ref.ID]
		if !ok {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "missing from candidate run")
			verdicts = append(verdicts, verdict)
			continue
		}
		verdict.CandSHA = cand.SHA256
		if ref.SHA256 != cand.SHA256 {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "input bytes differ")
		}
		if ref.Version != cand.Version {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("version %d vs %d", ref.Version, cand.Version))
		}
		if ok, why := sameFailure(ref.ParseErr, cand.ParseErr); !ok {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "parse error: "+why)
		}
		if ok, why := sameFailure(ref.ScanErr, cand.ScanErr); !ok {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "scan error: "+why)
		}
		if len(ref.Stmts) != len(cand.Stmts) {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("stmts %d vs %d", len(ref.Stmts), len(cand.Stmts)))
		} else {
			for i := range ref.Stmts {
				if ref.Stmts[i] != cand.Stmts[i] {
					verdict.Match = false
					verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("stmt %d %+v vs %+v", i, ref.Stmts[i], cand.Stmts[i]))
				}
			}
		}
		if len(ref.Tokens) != len(cand.Tokens) {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("tokens %d vs %d", len(ref.Tokens), len(cand.Tokens)))
		} else {
			for i := range ref.Tokens {
				if ref.Tokens[i] != cand.Tokens[i] {
					verdict.Match = false
					verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("token %d %+v vs %+v", i, ref.Tokens[i], cand.Tokens[i]))
					if len(verdict.Diffs) > 6 {
						verdict.Diffs = append(verdict.Diffs, "...")
						break
					}
				}
			}
		}
		verdict.RawParseEq = ref.RawParse == cand.RawParse
		verdict.RawScanEq = ref.RawScan == cand.RawScan
		if !verdict.RawParseEq {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "raw parse JSON differs")
		}
		if !verdict.RawScanEq {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, "raw scan JSON differs")
		}
		rp, cp := paramsOf(ref), paramsOf(cand)
		verdict.ParamsMatch = len(rp) == len(cp)
		if verdict.ParamsMatch {
			for i := range rp {
				if rp[i] != cp[i] {
					verdict.ParamsMatch = false
					break
				}
			}
		}
		if !verdict.ParamsMatch {
			verdict.Match = false
			verdict.Diffs = append(verdict.Diffs, fmt.Sprintf("params %v vs %v", rp, cp))
		}
		if ref.ParseErr != nil && cand.ParseErr != nil && ref.ParseErr.Type != cand.ParseErr.Type {
			verdict.TypeNote = fmt.Sprintf("error Go types differ (%s vs %s) with equal message/cursor", ref.ParseErr.Type, cand.ParseErr.Type)
		}
		if len(verdict.Diffs) == 0 {
			verdict.Diffs = []string{}
		}
		verdicts = append(verdicts, verdict)
	}
	matched := 0
	for _, verdict := range verdicts {
		if verdict.Match {
			matched++
		}
	}
	report := map[string]any{"cases": verdicts, "matched": matched, "total": len(verdicts)}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if out != "" {
		if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
			panic(err)
		}
	}
	fmt.Fprintf(os.Stderr, "verdict: %d/%d cases match\n", matched, len(verdicts))
	if matched != len(verdicts) {
		os.Exit(1)
	}
}

func main() {
	pin := flag.String("pin", "", "reference run to derive pins from")
	checkPins := flag.String("check", "", "pins file to verify a candidate against")
	candidate := flag.String("candidate", "", "candidate run for --check/--verdict")
	ref := flag.String("ref", "", "reference run for --verdict")
	verdict := flag.Bool("verdict", false, "compare two runs case by case")
	out := flag.String("out", "", "output file for --pin/--verdict")
	flag.Parse()

	switch {
	case *pin != "" && *out != "":
		pinMode(*pin, *out)
	case *checkPins != "" && *candidate != "":
		checkMode(*checkPins, *candidate)
	case *verdict && *ref != "" && *candidate != "":
		verdictMode(*ref, *candidate, *out)
	default:
		panic("usage: --pin ref --out pins | --check pins --candidate run | --verdict --ref a --candidate b [--out verdict]")
	}
}
