// Durable correction and interval publication.
//
// Every mutation the observer child accepts is appended as one JSON line
// to an append-only journal file and fsynced before the response goes
// out. A crash (SIGKILL, host reboot) can therefore lose at most the
// in-flight mutation, never an acknowledged one. The owner replays the
// journal after observer loss: acknowledged corrections and ticks come
// back, and any truncation, corruption, or sequence gap marks the
// remainder unknown instead of inventing facts.
package browser_observer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// journalHeader binds one journal file to its owned scope and launch.
// It is line zero; replay refuses a journal without an exact header.
type journalHeader struct {
	Header   bool     `json:"header"`
	Owner    Owner    `json:"owner"`
	Scope    string   `json:"scope"`
	Channels []string `json:"channels"`
	LaunchID string   `json:"launchId"`
	Limits   Limits   `json:"limits"`
}

// journalEntry is one acknowledged mutation, sequenced from 1 with no
// gaps. Only journaledOp operations are ever written.
type journalEntry struct {
	Seq     int    `json:"seq"`
	Op      string `json:"op"`
	Channel string `json:"channel,omitempty"`
	State   string `json:"state,omitempty"`
	Tick    int    `json:"tick,omitempty"`
	After   string `json:"after,omitempty"`
}

// Journal is an open append-only publication file.
type Journal struct {
	path   string
	file   *os.File
	next   int
	headed bool
}

// CreateJournal creates a fresh journal file. The path must not exist:
// publication never appends to a journal it did not start, so a stale
// file cannot contaminate a new interval.
func CreateJournal(path string) (*Journal, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("browser_observer: create journal: %w", err)
	}
	return &Journal{path: path, file: file, next: 1}, nil
}

// WriteHeader writes line zero. It must precede every entry and run
// exactly once.
func (j *Journal) WriteHeader(h journalHeader) error {
	if j.headed || j.next != 1 {
		return fmt.Errorf("browser_observer: journal header is not first")
	}
	h.Header = true
	line, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("browser_observer: encode journal header: %w", err)
	}
	if _, err := j.file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("browser_observer: write journal header: %w", err)
	}
	if err := j.file.Sync(); err != nil {
		return fmt.Errorf("browser_observer: sync journal header: %w", err)
	}
	j.headed = true
	return nil
}

// Append durably publishes one acknowledged mutation.
func (j *Journal) Append(op string, channel, state string, tick int, after string) error {
	if !j.headed {
		return fmt.Errorf("browser_observer: journal header is missing")
	}
	if !journaledOp(op) {
		return fmt.Errorf("browser_observer: op %q is not journaled", op)
	}
	entry := journalEntry{Seq: j.next, Op: op, Channel: channel, State: state, Tick: tick, After: after}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("browser_observer: encode journal entry: %w", err)
	}
	if _, err := j.file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("browser_observer: write journal entry: %w", err)
	}
	if err := j.file.Sync(); err != nil {
		return fmt.Errorf("browser_observer: sync journal entry: %w", err)
	}
	j.next++
	return nil
}

// Close closes the journal file.
func (j *Journal) Close() error {
	if err := j.file.Close(); err != nil {
		return fmt.Errorf("browser_observer: close journal: %w", err)
	}
	return nil
}

// Path reports the journal file path.
func (j *Journal) Path() string { return j.path }

// ReplayResult names how far the journal replayed. Truncated is true
// when a torn tail, corrupt line, sequence gap, or unappliable entry
// stopped the replay; every entry before the stop re-applied exactly.
type ReplayResult struct {
	Header    journalHeader
	Entries   int
	Truncated bool
}

// Replay rebuilds the engine from a durable journal after observer loss.
// It returns the rebuilt engine and its launch token. The caller must
// still mark observer loss: replay recovers acknowledged facts, and the
// loss itself makes the interval unknown.
func Replay(path string) (*Observer, string, ReplayResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: open journal: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), MaxWireLine)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: read journal header: %w", err)
		}
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: journal is empty")
	}
	var header journalHeader
	if err := strictDecode(scanner.Bytes(), &header); err != nil || !header.Header {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: journal header is corrupt")
	}
	engine, err := New([]Grant{{Owner: header.Owner, Scope: header.Scope, Channels: header.Channels}}, header.Limits)
	if err != nil {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: replay grants: %w", err)
	}
	if _, err := engine.OpenScope(header.Owner, header.Scope); err != nil {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: replay open scope: %w", err)
	}
	if _, err := engine.AdmitLaunch(header.Owner, header.Scope, header.LaunchID); err != nil {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: replay admit launch: %w", err)
	}
	token, err := engine.TokenForTest(header.Owner, header.Scope, header.LaunchID)
	if err != nil {
		return nil, "", ReplayResult{}, fmt.Errorf("browser_observer: replay token: %w", err)
	}
	result := ReplayResult{Header: header}
	want := 1
	truncated := false
	for scanner.Scan() {
		var entry journalEntry
		if err := strictDecode(scanner.Bytes(), &entry); err != nil {
			truncated = true
			break
		}
		if entry.Seq != want || !journaledOp(entry.Op) {
			truncated = true
			break
		}
		if err := applyReplay(engine, header, token, entry); err != nil {
			truncated = true
			break
		}
		want++
		result.Entries++
	}
	if err := scanner.Err(); err != nil {
		truncated = true
	}
	result.Truncated = truncated
	return engine, token, result, nil
}

func strictDecode(line []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func applyReplay(engine *Observer, header journalHeader, token string, entry journalEntry) error {
	owner, scope, launch := header.Owner, header.Scope, header.LaunchID
	var err error
	switch entry.Op {
	case OpSample:
		_, err = engine.Observe(owner, scope, launch, token, entry.Channel, entry.State)
	case OpEndTick:
		_, err = engine.EndTick(owner, scope, launch, token)
	case OpNoteDriverDeath:
		err = engine.NoteDriverDeath(owner, scope, launch, token)
	case OpPublishCorr:
		_, err = engine.PublishCorrection(owner, scope, launch, token, entry.Tick, entry.Channel, entry.After)
	case OpMarkUnknown:
		_, err = engine.MarkUnknown(owner, scope, launch, token, entry.Tick, entry.Channel)
	case OpSealDisposal:
		_, err = engine.SealDisposal(owner, scope, launch, token)
	case OpInterrupt:
		err = engine.InterruptScope(owner, scope)
	case OpResume:
		err = engine.ResumeScope(owner, scope)
	default:
		err = fmt.Errorf("not a journaled op")
	}
	return err
}
