package reference

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Build runs the ordinary reviewed bootstrap (tools/distbuild, verified
// drift-free by DistbuildDriftFree) against source with the pinned archive,
// staging the seed distribution under outParent. repoRoot is the current
// checkout whose distbuild runs; source is the pristine selection checkout.
// It returns the staged version-root path. The build never invokes the
// candidate compiler; distribution.Build only compiles it.
func Build(ctx context.Context, repoRoot, source, archive, outParent, version string) (string, error) {
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("reference: invalid seed version %q", version)
	}
	for name, dir := range map[string]string{"repo": repoRoot, "source": source} {
		info, err := os.Stat(dir)
		if err != nil {
			return "", fmt.Errorf("reference: %s dir: %w", name, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("reference: %s is not a directory: %s", name, dir)
		}
	}
	if _, err := os.Stat(archive); err != nil {
		return "", fmt.Errorf("reference: archive: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "run", "./tools/distbuild",
		"--archive", archive, "--out", outParent, "--source", source, "--version", version)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("reference: distbuild: %v\n%s", err, exit.Stderr)
		}
		return "", fmt.Errorf("reference: distbuild: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("reference: distbuild printed no path")
	}
	if !filepath.IsAbs(root) {
		root = filepath.Join(repoRoot, root)
	}
	return root, nil
}
