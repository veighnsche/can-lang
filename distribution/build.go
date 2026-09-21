package distribution

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Build consumes an already acquired archive. It never downloads or overwrites
// an existing version; publication happens only after every asset and launcher
// has been assembled in a sibling staging directory.
func Build(ctx context.Context, source, output, archive, version string) (string, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`).MatchString(version) {
		return "", fmt.Errorf("invalid distribution version")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return "", err
	}
	target := PinnedTarget()
	final := filepath.Join(output, "can-"+version+"-"+target.TargetID)
	if _, err := os.Lstat(final); !os.IsNotExist(err) {
		return "", fmt.Errorf("distribution already exists or cannot be inspected: %s", final)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		return "", err
	}
	if int64(len(data)) != target.Upstream.Size || Hash(data) != target.Upstream.SHA256 {
		return "", fmt.Errorf("upstream archive size or SHA-256 mismatch")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var runtimeBytes []byte
	for _, file := range zr.File {
		if file.Name != target.Upstream.Member {
			continue
		}
		if runtimeBytes != nil {
			return "", fmt.Errorf("duplicate runtime archive member")
		}
		r, err := file.Open()
		if err != nil {
			return "", err
		}
		runtimeBytes, err = io.ReadAll(r)
		r.Close()
		if err != nil {
			return "", err
		}
	}
	if Hash(runtimeBytes) != target.Runtime.SHA256 {
		return "", fmt.Errorf("extracted runtime SHA-256 mismatch")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(output, ".can-stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	manifest := Manifest{SchemaVersion: 1, Kind: "can.development-distribution", Version: version, TargetID: target.TargetID, Files: map[string]string{}}
	// macOS layouts must not acquire case-aliasing assets, even when the build
	// checkout happens to live on a case-sensitive volume. Generated destinations
	// are claimed before source assets are copied; no later write may replace one.
	written := map[string]bool{"manifest.json": true, "bin/canlc": true}
	write := func(name string, data []byte, mode os.FileMode) error {
		key := strings.ToLower(filepath.ToSlash(name))
		if written[key] {
			return fmt.Errorf("bundle destination collision: %s", name)
		}
		path := filepath.Join(stage, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			return fmt.Errorf("create unique bundle asset %s: %w", name, err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		written[key] = true
		manifest.Files[name] = Hash(data)
		return nil
	}
	if err := write(target.Runtime.Executable, runtimeBytes, 0755); err != nil {
		return "", err
	}
	if err := write("distribution/target.json", TargetJSON, 0644); err != nil {
		return "", err
	}
	config, err := os.ReadFile(filepath.Join(source, "tools/runtime/tsconfig.json"))
	if err != nil {
		return "", err
	}
	if err := write("tsconfig.json", config, 0644); err != nil {
		return "", err
	}
	for _, dir := range []string{"runtime", "tools/runtime", "distribution/assets", "distribution/notices"} {
		err := filepath.WalkDir(filepath.Join(source, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("bundle asset is not a regular file: %s", path)
			}
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return write(filepath.ToSlash(rel), data, 0644)
		})
		if err != nil {
			return "", err
		}
	}
	if err := VerifyHTMX(source); err != nil {
		return "", err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(stage, "manifest.json"), manifestBytes, 0644); err != nil {
		return "", err
	}
	if err := os.Mkdir(filepath.Join(stage, "bin"), 0755); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags", "-X main.version="+version+" -X main.bundleManifestSHA256="+Hash(manifestBytes), "-o", filepath.Join(stage, "bin/canlc"), "./compiler")
	cmd.Dir = source
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		switch key {
		case "GOOS", "GOARCH", "CGO_ENABLED", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN":
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "GOOS=darwin", "GOARCH=arm64", "CGO_ENABLED=0", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build launcher: %w\n%s", err, output)
	}
	if err := os.Rename(stage, final); err != nil {
		return "", err
	}
	return final, nil
}
