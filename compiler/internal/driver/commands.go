package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
)

type BuildReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	BuildID       string `json:"buildID"`
	Directory     string `json:"directory"`
	Entry         string `json:"entry"`
}

func (r *Runtime) build(ctx context.Context, store *OutputStore) (BuildReport, error) {
	program, err := check.CheckProgram(store.Graph)
	if err != nil {
		return BuildReport{}, err
	}
	assets, err := r.PrivateArtifacts()
	if err != nil {
		return BuildReport{}, err
	}
	artifacts, err := emit.ProgramModules(program, assets.Directory, assets.Files)
	if err != nil {
		return BuildReport{}, err
	}
	launcher, err := regularFile(r.Root, "bin/canlc", true)
	if err != nil {
		return BuildReport{}, err
	}
	options, _ := json.Marshal(struct {
		Schema int
		Entry  string
	}{1, "main"})
	inputs := store.BuildInputs(hashBytes(launcher), catalogue.SourceHash(), assets.Identity, hashBytes(options))
	prepared, err := PrepareOutput(inputs, "entry.ts", artifacts)
	if err != nil {
		return BuildReport{}, err
	}
	if err = r.ValidateOutput(ctx, prepared); err != nil {
		return BuildReport{}, err
	}
	directory, err := store.Publish(prepared)
	if err != nil {
		return BuildReport{}, err
	}
	return BuildReport{SchemaVersion: 1, Kind: "can.build", BuildID: prepared.BuildID(), Directory: directory, Entry: "entry.ts"}, nil
}
func (r *Runtime) Build(ctx context.Context, projectDirectory string) (BuildReport, error) {
	if r == nil {
		return BuildReport{}, fmt.Errorf("build requires a bundled runtime")
	}
	store, err := BeginOutput(projectDirectory)
	if err != nil {
		return BuildReport{}, err
	}
	defer store.Close()
	return r.build(ctx, store)
}
