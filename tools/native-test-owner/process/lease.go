package process

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
)

// Lease errors. Callers distinguish them with errors.Is.
var (
	ErrLeaseHeld    = errors.New("process: lease still held")
	ErrLeaseNotHeld = errors.New("process: lease not held")
)

// Lease is one kernel-level shared lock on a lease file. The lock lives
// in the open file description: forked children that inherit the
// descriptor share it, so protection survives the launcher closing its
// own copy while any descendant still holds the file open. Only closing
// every copy releases the lock.
type Lease struct {
	path string
	mu   sync.Mutex
	file *os.File
}

// AcquireLease creates (or opens) the lease file at path and takes a
// non-blocking shared lock on it. A conflicting exclusive lock fails
// here instead of blocking.
func AcquireLease(path string) (*Lease, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty lease path", ErrInvalid)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("process: open lease: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("process: acquire lease: %w", err)
	}
	return &Lease{path: path, file: f}, nil
}

// Path reports the lease file path.
func (l *Lease) Path() string { return l.path }

// File exposes the locked descriptor for inheritance across spawn. The
// caller must keep the Lease open until the child that inherits it has
// been reaped; closing early releases the launcher's share.
func (l *Lease) File() *os.File {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file
}

// Close releases the launcher's share of the lock. Protection continues
// while any inheriting descendant still holds its copy. Close is safe
// for concurrent use: Release, Abort and CloseLeaseEarly may race on
// the same share, and the losers observe an already-closed lease.
func (l *Lease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// ProbeLease is the independent release witness: it opens path afresh
// (no shared state with any owner or child descriptor) and attempts a
// non-blocking exclusive lock. It reports held=true while any process
// holds a shared or exclusive lock, and atomically releases its probe
// lock before returning held=false. Any process can run it.
func ProbeLease(path string) (held bool, err error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return false, fmt.Errorf("process: probe lease: %w", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return true, nil
		}
		return false, fmt.Errorf("process: probe lease: %w", err)
	}
	return false, nil
}
