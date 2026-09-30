package acceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrNAttest reports a failed N version attestation.
var ErrNAttest = errors.New("acceptance: N version attestation failed")

// MaxExecutableBytes caps the executable read during attestation.
// Subjects are test fixtures and toolchain binaries, never disk images.
const MaxExecutableBytes = 512 << 20

// NIdentity binds a subject name and explicit version facts to the exact
// executable bytes under test. The digest is computed over the resolved
// executable file so a replaced binary cannot ride an old attestation.
type NIdentity struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Executable string `json:"executable"`
	Digest     string `json:"digest"`
}

// AttestN attests a subject executable: the path must be absolute, the
// name and version facts must be explicit, and the digest binds the
// current file bytes.
func AttestN(name, version, executable string) (NIdentity, error) {
	if name == "" || version == "" {
		return NIdentity{}, fmt.Errorf("%w: subject needs an explicit name and version", ErrNAttest)
	}
	if executable == "" || !filepath.IsAbs(executable) {
		return NIdentity{}, fmt.Errorf("%w: executable must be an absolute path", ErrNAttest)
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return NIdentity{}, fmt.Errorf("%w: resolve executable: %v", ErrNAttest, err)
	}
	digest, err := digestFile(resolved)
	if err != nil {
		return NIdentity{}, fmt.Errorf("%w: %v", ErrNAttest, err)
	}
	return NIdentity{Name: name, Version: version, Executable: resolved, Digest: digest}, nil
}

// Verify re-hashes the attested executable and compares it against the
// bound digest. Any replacement, truncation or tampering fails.
func (id NIdentity) Verify() error {
	if id.Name == "" || id.Version == "" || id.Executable == "" || id.Digest == "" {
		return fmt.Errorf("%w: incomplete identity", ErrNAttest)
	}
	digest, err := digestFile(id.Executable)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNAttest, err)
	}
	if digest != id.Digest {
		return fmt.Errorf("%w: executable bytes no longer match %s", ErrNAttest, id.Digest)
	}
	return nil
}

func digestFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read executable: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, MaxExecutableBytes+1))
	if err != nil {
		return "", fmt.Errorf("read executable: %w", err)
	}
	if n > MaxExecutableBytes {
		return "", fmt.Errorf("executable exceeds bound")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
