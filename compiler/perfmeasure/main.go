// perfmeasure measures production compiler code without modifying it.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/driver"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type prepared struct {
	Artifacts       []ir.Artifact
	Assertions      []ir.Artifact
	Roots           []ir.AssertionRoot
	MappedArtifacts []ir.Artifact
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}()
	directory := flag.String("project", "", "fixture directory")
	manifest := flag.String("manifest", "", "qualified distribution manifest digest")
	mode := flag.String("mode", "compiler", "prepare, compiler, assertions, artifacts")
	loops := flag.Int("iterations", 1, "operations per sample")
	batches := flag.Int("batches", 3, "measured samples")
	warmups := flag.Int("warmups", 1, "excluded sample batches")
	target := flag.String("target", "bun", "bun or browser")
	phases := flag.String("phases", "", "optional comma-separated compiler phases")
	jobs := flag.Int("jobs", 1, "assertion worker count")
	flag.Parse()
	if *loops < 1 || *batches < 1 || *warmups < 0 {
		panic("invalid sample counts")
	}
	r, err := driver.Resolve(*manifest)
	must(err)
	private, err := r.PrivateArtifacts()
	must(err)
	checkProgram := func(g *project.Graph) (*check.Program, error) {
		if *target == "browser" {
			return check.CheckBrowserProgram(g)
		}
		if *target != "bun" {
			return nil, fmt.Errorf("unknown target: %s", *target)
		}
		return check.CheckProgram(g)
	}
	emitProgram := func(p *check.Program) ([]ir.Artifact, error) {
		if *target == "browser" {
			return emit.BrowserModules(p, private.Directory, private.Files)
		}
		return emit.ProgramModules(p, private.Directory, private.Files)
	}
	g, err := project.Load(*directory)
	must(err)
	if *mode == "prepare" {
		p, err := checkProgram(g)
		must(err)
		a, err := emitProgram(p)
		must(err)
		// CheckProgram already checks and retains all concrete assertions.
		// Reuse its checked IR during untimed preparation; no checks are skipped.
		ap := p
		aa, err := emit.AssertionModules(ap, private.Directory, private.Files)
		must(err)
		aa, err = encodeMappedArtifacts(r, ap, aa)
		must(err)
		roots := make([]ir.AssertionRoot, len(ap.Assertions))
		for i, t := range ap.Assertions {
			roots[i] = t.Root
		}
		mapped, err := encodeMappedArtifacts(r, p, a)
		must(err)
		raw, err := json.Marshal(prepared{Artifacts: a, Assertions: aa, Roots: roots, MappedArtifacts: mapped})
		must(err)
		must(os.WriteFile(filepath.Join(*directory, "prepared.json"), raw, 0600))
		return
	}
	var saved prepared
	raw, err := os.ReadFile(filepath.Join(*directory, "prepared.json"))
	must(err)
	must(json.Unmarshal(raw, &saved))
	metrics := map[string]any{"artifact_files": len(saved.Artifacts), "assertion_roots": len(saved.Roots)}
	total := 0
	for _, a := range saved.Artifacts {
		total += len(a.Bytes)
	}
	metrics["artifact_bytes"] = total
	sourceCount, sourceBytes := 0, 0
	for _, project := range g.Projects {
		for _, src := range project.Sources {
			sourceCount++
			sourceBytes += len(src.Bytes)
		}
	}
	metrics["source_files"] = sourceCount
	metrics["source_bytes"] = sourceBytes
	metrics["projects"] = len(g.Projects)
	metrics["target"] = *target
	works := map[string]func() error{}
	scopes := map[string]string{}
	checks := map[string]func() error{}
	switch *mode {
	case "compiler":
		w, err := resolve.Build(g)
		must(err)
		p, err := checkProgram(g)
		must(err)
		works["parse"] = func() error {
			for _, p := range g.Projects {
				for _, s := range p.Sources {
					f, e := source.New(s.Name, string(s.Bytes))
					if e != nil {
						return e
					}
					result := syntax.Parse(f)
					if !result.OK() {
						return fmt.Errorf("parse diagnostics: %v", result.Diagnostics)
					}
				}
			}
			return nil
		}
		works["load"] = func() error { _, e := project.Load(*directory); return e }
		works["resolve"] = func() error { _, e := resolve.Build(g); return e }
		works["declarations"] = func() error { _, e := types.CheckDeclarations(w); return e }
		works["check"] = func() error { _, e := checkProgram(g); return e }
		var lastArtifacts []ir.Artifact
		works["emit"] = func() error {
			var e error
			lastArtifacts, e = emitProgram(p)
			return e
		}
		verifyEmission := func() error {
			if len(lastArtifacts) != len(saved.Artifacts) {
				return fmt.Errorf("artifact count changed")
			}
			for i, a := range lastArtifacts {
				if a.Path != saved.Artifacts[i].Path || !bytes.Equal(a.Bytes, saved.Artifacts[i].Bytes) {
					return fmt.Errorf("prepared emission mismatch: %s", a.Path)
				}
			}
			return nil
		}
		checks["emit"] = verifyEmission
		checks["pipeline"] = verifyEmission
		works["pipeline"] = func() error {
			g, e := project.Load(*directory)
			if e != nil {
				return e
			}
			p, e := checkProgram(g)
			if e != nil {
				return e
			}
			lastArtifacts, e = emitProgram(p)
			return e
		}
		scopes["parse"] = "source.New + syntax.Parse for all resident source bytes"
		scopes["load"] = "project.Load, including filesystem reads and parse"
		scopes["resolve"] = "resolve.Build on resident parsed graph"
		scopes["declarations"] = "types.CheckDeclarations on resident resolved world"
		scopes["check"] = "target-specific check.CheckProgram / check.CheckBrowserProgram on parsed graph, including resolution, declaration/body checking and IR"
		scopes["emit"] = "target-specific emit.ProgramModules / emit.BrowserModules on resident checked IR and runtime artifact bytes"
		scopes["pipeline"] = "project.Load + target-specific checking + module emission; excludes TypeScript validation, assertions and publication"
	case "assertions", "artifacts":
		store, e := driver.BeginOutput(*directory)
		must(e)
		defer store.Close()
		inputs := store.BuildInputs(private.Identity, private.Identity, private.Identity, private.Identity)
		if *mode == "assertions" {
			metrics["artifact_files"] = len(saved.Assertions)
			assertionBytes := 0
			for _, a := range saved.Assertions {
				assertionBytes += len(a.Bytes)
			}
			metrics["artifact_bytes"] = assertionBytes
			output, e := driver.PrepareOutput(inputs, "entry.ts", saved.Assertions)
			must(e)
			must(r.ValidateOutput(context.Background(), output))
			id, _, e := store.Stage(output)
			must(e)
			leases := make([]*driver.OutputLease, *loops*(*warmups+*batches))
			for i := range leases {
				leases[i], e = store.AcquireGeneration(id)
				must(e)
				defer leases[i].Close()
			}
			leaseIndex := 0
			must(store.Close())
			works["supervised-roots"] = func() error {
				lease := leases[leaseIndex]
				leaseIndex++
				entries, e := r.RunSupervised(context.Background(), lease, saved.Roots, os.Environ(), nil, 5000, io.Discard, *jobs)
				if e != nil {
					return e
				}
				if len(entries) != len(saved.Roots) {
					return fmt.Errorf("incomplete assertion roots")
				}
				for _, entry := range entries {
					metrics["root_wall_ms"] = append(metrics["root_wall_ms"].([]any), map[string]any{"root": entry["root"], "elapsed_ms": entry["elapsedMs"]})
					if entry["passed"] != true {
						return fmt.Errorf("assertion failed: %v", entry)
					}
				}
				return nil
			}
			scopes["supervised-roots"] = "RunSupervised on precompiled leased generation: generation validation, bounded worker launches, real assertion harness, report delivery and reaping; compilation/staging excluded"
			metrics["jobs"] = *jobs
			metrics["root_wall_ms"] = []any{}
		} else {
			output, e := driver.PrepareOutput(inputs, "entry.ts", saved.MappedArtifacts)
			must(e)
			must(r.ValidateOutput(context.Background(), output))
			works["validate"] = func() error {
				p, e := driver.PrepareOutput(inputs, "entry.ts", saved.MappedArtifacts)
				if e != nil {
					return e
				}
				return r.ValidateOutput(context.Background(), p)
			}
			scopes["validate"] = "PrepareOutput inventory/content hashing + production ValidateOutput native TypeScript transpilation, source-index/map relationships and structural checks on resident mapped emitted artifacts"
			_, e = store.Publish(output)
			must(e)
			works["publish-reuse"] = func() error { _, e := store.Publish(output); return e }
			scopes["publish-reuse"] = "OutputStore.Publish identical validated generation, including filesystem integrity checks and current selection; no new payload writes"
			p, e := checkProgram(g)
			must(e)
			var lastMaps []ir.Artifact
			works["source-maps"] = func() error { var err error; lastMaps, err = encodeMappedArtifacts(r, p, saved.Artifacts); return err }
			scopes["source-maps"] = "checked IR span/index assembly + qualified production source-maps.ts encoding and map validation/receipt; filesystem publication excluded"
			checks["source-maps"] = func() error { return sameArtifacts(lastMaps, saved.MappedArtifacts) }
			metrics["mapped_artifact_files"] = len(saved.MappedArtifacts)
			mappedBytes := 0
			for _, a := range saved.MappedArtifacts {
				mappedBytes += len(a.Bytes)
			}
			metrics["mapped_artifact_bytes"] = mappedBytes
			metrics["artifact_files"] = len(saved.MappedArtifacts)
			metrics["artifact_bytes"] = mappedBytes
			// Each destination starts with an owned but empty output store.
			// Project copying, snapshot loading, locks and native validation are untimed.
			fresh := make([]*driver.OutputStore, *loops*(*warmups+*batches))
			for i := range fresh {
				destination, e := os.MkdirTemp(filepath.Dir(*directory), "publish-flat-")
				must(e)
				defer func() { must(os.RemoveAll(destination)) }()
				must(copyFlatInputs(*directory, destination, g))
				fresh[i], e = driver.BeginOutput(destination)
				must(e)
				defer fresh[i].Close()
				if fresh[i].BuildInputs(private.Identity, private.Identity, private.Identity, private.Identity) != inputs {
					panic("new publication input identity changed")
				}
			}
			publishIndex := 0
			published := []string{}
			publishedStores := []*driver.OutputStore{}
			works["publish-new"] = func() error {
				store := fresh[publishIndex]
				path, e := store.Publish(output)
				publishIndex++
				if e == nil {
					published = append(published, path)
					publishedStores = append(publishedStores, store)
				}
				return e
			}
			scopes["publish-new"] = "OutputStore.Publish into a fresh owned empty generation/object store: payload writes, content identity storage, generation manifest and current selection; fixture copying/locking/snapshot loading and native validation excluded"
			checks["publish-new"] = func() error {
				for i, dir := range published {
					lease, e := publishedStores[i].AcquireCurrent()
					if e != nil {
						return e
					}
					selected := lease.Manifest.BuildID == output.BuildID() && lease.Directory == dir
					if e = lease.Close(); e != nil {
						return e
					}
					if !selected {
						return fmt.Errorf("new current selection differs")
					}
					for _, a := range saved.MappedArtifacts {
						data, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(a.Path)))
						if e != nil {
							return e
						}
						if !bytes.Equal(data, a.Bytes) {
							return fmt.Errorf("published bytes differ: %s", a.Path)
						}
					}
				}
				published = nil
				publishedStores = nil
				return nil
			}
		}
	default:
		panic("unknown mode")
	}
	order := []string{"parse", "load", "resolve", "declarations", "check", "emit", "pipeline", "supervised-roots", "source-maps", "validate", "publish-new", "publish-reuse"}
	enc := json.NewEncoder(os.Stdout)
	for _, name := range order {
		if *phases != "" && !strings.Contains(","+*phases+",", ","+name+",") {
			continue
		}
		work, ok := works[name]
		if !ok {
			continue
		}
		for batch := 0; batch < *warmups+*batches; batch++ {
			if *mode == "assertions" {
				metrics["root_wall_ms"] = []any{}
			}
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			for i := 0; i < *loops; i++ {
				must(work())
			}
			elapsed := time.Since(start)
			runtime.ReadMemStats(&after)
			if verify, ok := checks[name]; ok {
				must(verify())
			}
			must(enc.Encode(map[string]any{"name": name, "warmup": batch < *warmups, "ns_per_op": float64(elapsed.Nanoseconds()) / float64(*loops), "alloc_bytes_per_op": float64(after.TotalAlloc-before.TotalAlloc) / float64(*loops), "allocations_per_op": float64(after.Mallocs-before.Mallocs) / float64(*loops), "gc_cycles": after.NumGC - before.NumGC, "timing_scope": scopes[name], "metrics": metrics}))
		}
	}
}
