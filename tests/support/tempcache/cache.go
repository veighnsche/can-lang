// Package tempcache owns disposable test bundle caches and recovers abandoned
// caches from earlier interrupted suites. It never removes legacy cache roots.
package tempcache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const prefix = "can-suite-cache-"
const markerName = ".can-test-owner.json"
const leaseName = ".can-test-lease"
const schema = "can.test-suite-cache/1"

// Recovery waits beyond the harness's 30-minute fill deadline. Age alone
// never establishes inactivity: recovery also requires a dead owner, an
// uncontended lease, and a successful process/open-file liveness snapshot.
const recoveryAge = time.Hour
const recoveryLimit = 16

type owner struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	PID    int    `json:"pid"`
}

type Cache struct {
	Path  string
	lease *os.File
	info  os.FileInfo
}

// Open creates a unique child under parent (the system temporary directory when
// empty). The supplied parent remains caller-owned. Only marked, old, unlocked
// children with dead owners are candidates for next-run recovery.
func Open(parent string) (*Cache, error) {
	if parent == "" {
		parent = os.TempDir()
	}
	parent, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(parent, 0755); err != nil {
		return nil, err
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, err
	}
	if err := recoverAbandoned(parent, time.Now(), noReferences); err != nil {
		return nil, err
	}
	path, err := os.MkdirTemp(parent, prefix)
	if err != nil {
		return nil, err
	}
	c := &Cache{Path: path}
	c.info, err = os.Lstat(path)
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if failed {
			c.Close()
		}
	}()
	c.lease, err = os.OpenFile(filepath.Join(path, leaseName), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(c.lease.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, err
	}
	data, err := json.Marshal(owner{Schema: schema, Name: filepath.Base(path), PID: os.Getpid()})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(path, markerName), data, 0600); err != nil {
		return nil, err
	}
	failed = false
	return c, nil
}

// Close removes the owned child while its lease is still held. Directory
// replacement is refused rather than deleting a different owner's path.
func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	var err error
	if c.Path != "" {
		current, statErr := os.Lstat(c.Path)
		if os.IsNotExist(statErr) {
			// Already removed; nothing else at this path belongs to us.
		} else if statErr != nil {
			err = statErr
		} else if current.Mode()&os.ModeSymlink != 0 || !os.SameFile(c.info, current) {
			err = fmt.Errorf("suite cache directory was replaced: %s", c.Path)
		} else {
			err = os.RemoveAll(c.Path)
		}
		c.Path = ""
	}
	if c.lease != nil {
		err = errors.Join(err, c.lease.Close())
		c.lease = nil
	}
	return err
}

type abandoned struct {
	path  string
	info  os.FileInfo
	lease *os.File
	root  *os.Root
}

func recoverAbandoned(parent string, now time.Time, check func([]string) error) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	var candidates []*abandoned
	defer func() {
		for _, c := range candidates {
			c.lease.Close()
			c.root.Close()
		}
	}()
	inspected := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if inspected == recoveryLimit {
			break
		}
		inspected++
		if c := recoverOne(filepath.Join(parent, entry.Name()), now); c != nil {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	paths := make([]string, len(candidates))
	for i, c := range candidates {
		paths[i] = c.path
	}
	if err := check(paths); err != nil {
		fmt.Fprintf(os.Stderr, "suite cache recovery skipped (%d candidates): %v\n", len(candidates), err)
		return nil
	}
	for _, c := range candidates {
		current, err := os.Lstat(c.path)
		if err != nil || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(c.info, current) {
			continue
		}
		if err := os.RemoveAll(c.path); err != nil {
			return err
		}
	}
	return nil
}

// noReferences checks all candidates with one ps and one lsof snapshot, without
// recursive directory scans. lsof covers this user, who owns every candidate.
// Missing tools, partial output, warnings, timeout,
// or any reference cause recovery to skip rather than guessing inactivity.
func noReferences(paths []string) error {
	for _, command := range [][]string{{"ps", "-axo", "command="}, {"lsof", "-n", "-P", "-u", strconv.Itoa(os.Getuid()), "-Fpn"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		output := &limitedOutput{}
		stderr := &limitedOutput{}
		cmd.Stdout, cmd.Stderr = output, stderr
		err := cmd.Run()
		cancel()
		if err != nil || stderr.Len() != 0 || output.overflow || stderr.overflow || output.Len() == 0 {
			return fmt.Errorf("%s liveness unavailable or incomplete", command[0])
		}
		for _, path := range paths {
			if strings.Contains(output.String(), path) {
				return fmt.Errorf("%s reports an active reference", command[0])
			}
		}
	}
	return nil
}

type limitedOutput struct {
	bytes.Buffer
	overflow bool
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	if w.Len()+len(p) > 8<<20 {
		w.overflow = true
		return 0, fmt.Errorf("liveness output exceeds limit")
	}
	return w.Buffer.Write(p)
}

func recoverOne(path string, now time.Time) *abandoned {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || now.Sub(info.ModTime()) < recoveryAge {
		return nil
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) {
		return nil
	}
	// O_NOFOLLOW rejects marker/lease symlinks. OpenRoot keeps their lookup
	// inside this candidate even if an intermediate pathname is replaced.
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil
	}
	keep := false
	defer func() {
		if !keep {
			root.Close()
		}
	}()
	marker, err := root.OpenFile(markerName, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil
	}
	defer marker.Close()
	markerInfo, err := marker.Stat()
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Size() > 4096 {
		return nil
	}
	var owned owner
	decoder := json.NewDecoder(marker)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&owned) != nil || owned.Schema != schema || owned.Name != filepath.Base(path) || owned.PID <= 0 {
		return nil
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil
	}
	// Unknown permission/process errors mean potentially live; do not reclaim.
	if !errors.Is(syscall.Kill(owned.PID, 0), syscall.ESRCH) {
		return nil
	}
	lease, err := root.OpenFile(leaseName, os.O_RDWR|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil
	}
	defer func() {
		if !keep {
			lease.Close()
		}
	}()
	leaseInfo, err := lease.Stat()
	if err != nil || !leaseInfo.Mode().IsRegular() {
		return nil
	}
	if syscall.Flock(int(lease.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return nil
	}
	current, err := os.Lstat(path)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, current) {
		return nil
	}
	keep = true
	return &abandoned{path: path, info: info, lease: lease, root: root}
}
