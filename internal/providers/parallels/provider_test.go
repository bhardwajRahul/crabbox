package parallels

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	core "github.com/openclaw/crabbox/internal/cli"
	"github.com/openclaw/crabbox/internal/providers/shared"
)

func TestApplyFlagsNameOverridesClearIDOverrides(t *testing.T) {
	cfg := core.BaseConfig()
	cfg.Provider = "parallels"
	cfg.Parallels.SourceID = "old-source-id"
	cfg.Parallels.SourceSnapshotID = "old-snapshot-id"

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	provider := Provider{}
	values := provider.RegisterFlags(fs, cfg)
	if err := fs.Parse([]string{
		"--parallels-source", "Ubuntu 25.10",
		"--parallels-source-snapshot", "fresh",
	}); err != nil {
		t.Fatal(err)
	}
	if err := provider.ApplyFlags(&cfg, fs, values); err != nil {
		t.Fatal(err)
	}
	if cfg.Parallels.Source != "Ubuntu 25.10" || cfg.Parallels.SourceID != "" {
		t.Fatalf("source override not applied cleanly: %#v", cfg.Parallels)
	}
	if cfg.Parallels.SourceSnapshot != "fresh" || cfg.Parallels.SourceSnapshotID != "" {
		t.Fatalf("snapshot override not applied cleanly: %#v", cfg.Parallels)
	}
}

func TestApplyFlagsKeepsExplicitTargetOverTemplate(t *testing.T) {
	cfg := core.BaseConfig()
	cfg.Provider = "parallels"
	cfg.TargetOS = core.TargetLinux
	cfg.WindowsMode = core.WindowsModeNormal
	cfg.Parallels.Templates = map[string]core.ParallelsTemplateConfig{
		"win": {
			TargetOS:    core.TargetWindows,
			WindowsMode: core.WindowsModeWSL2,
			Source:      "Windows 11",
		},
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("target", "", "")
	fs.String("windows-mode", "", "")
	provider := Provider{}
	values := provider.RegisterFlags(fs, cfg)
	if err := fs.Parse([]string{
		"--target", "linux",
		"--windows-mode", "normal",
		"--parallels-template", "win",
	}); err != nil {
		t.Fatal(err)
	}
	if err := provider.ApplyFlags(&cfg, fs, values); err != nil {
		t.Fatal(err)
	}
	if cfg.TargetOS != core.TargetLinux || cfg.WindowsMode != core.WindowsModeNormal {
		t.Fatalf("explicit target flags should win over template: target=%s windowsMode=%s", cfg.TargetOS, cfg.WindowsMode)
	}
	if cfg.Parallels.Source != "Windows 11" {
		t.Fatalf("template source should still apply: %#v", cfg.Parallels)
	}
}

func TestApplyFlagsRejectsInvalidStartupTimeout(t *testing.T) {
	cfg := core.BaseConfig()
	cfg.Provider = "parallels"

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	provider := Provider{}
	values := provider.RegisterFlags(fs, cfg)
	if err := fs.Parse([]string{"--parallels-startup-timeout", "nope"}); err != nil {
		t.Fatal(err)
	}
	if err := provider.ApplyFlags(&cfg, fs, values); err == nil {
		t.Fatal("expected invalid startup timeout error")
	}
}

func TestResolveReportsPartialFleetInventory(t *testing.T) {
	backend := &leaseBackend{
		DirectSSHBackend: sharedBackend(testParallelsFleetConfig(), &parallelsFleetRunner{}),
	}
	_, err := backend.Resolve(context.Background(), ResolveRequest{ID: "missing-lease"})
	if err == nil {
		t.Fatal("Resolve err=nil, want partial fleet inventory error")
	}
	for _, want := range []string{"fleet inventory incomplete", "bad-host", "ssh failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err=%q missing %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "lease not found") {
		t.Fatalf("err=%q should not report false not-found", err)
	}
}

func TestListReportsPartialFleetInventory(t *testing.T) {
	backend := &leaseBackend{
		DirectSSHBackend: sharedBackend(testParallelsFleetConfig(), &parallelsFleetRunner{}),
	}
	leases, err := backend.List(context.Background(), ListRequest{})
	if err == nil {
		t.Fatalf("List err=nil leases=%#v, want partial fleet inventory error", leases)
	}
	for _, want := range []string{"fleet inventory incomplete", "bad-host", "ssh failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err=%q missing %q", err, want)
		}
	}
}

func TestCleanupStopsOnPartialFleetInventory(t *testing.T) {
	runner := &parallelsFleetRunner{}
	backend := &leaseBackend{
		DirectSSHBackend: sharedBackend(testParallelsFleetConfig(), runner),
	}
	err := backend.Cleanup(context.Background(), CleanupRequest{})
	if err == nil {
		t.Fatal("Cleanup err=nil, want partial fleet inventory error")
	}
	if runner.deleteCalls != 0 {
		t.Fatalf("deleteCalls=%d want 0 before complete inventory", runner.deleteCalls)
	}
}

func sharedBackend(cfg core.Config, runner core.CommandRunner) shared.DirectSSHBackend {
	return shared.DirectSSHBackend{Cfg: cfg, RT: Runtime{Exec: runner}}
}

func testParallelsFleetConfig() core.Config {
	cfg := core.BaseConfig()
	cfg.Provider = "parallels"
	cfg.TargetOS = core.TargetLinux
	cfg.Parallels.Hosts = []core.ParallelsHostConfig{
		{Name: "good-host", Host: "good.example"},
		{Name: "bad-host", Host: "bad.example"},
	}
	return cfg
}

type parallelsFleetRunner struct {
	deleteCalls int
}

func (r *parallelsFleetRunner) Run(_ context.Context, req core.LocalCommandRequest) (core.LocalCommandResult, error) {
	if req.Name != "ssh" || len(req.Args) < 2 {
		return core.LocalCommandResult{}, errors.New("unexpected command")
	}
	host := req.Args[len(req.Args)-2]
	remote := req.Args[len(req.Args)-1]
	if strings.Contains(remote, " delete ") {
		r.deleteCalls++
	}
	if host == "bad.example" {
		return core.LocalCommandResult{Stderr: "permission denied"}, errors.New("ssh failed")
	}
	return core.LocalCommandResult{Stdout: `[{"ID":"vm-good","Name":"crabbox-cbx-good-blue","State":"running","ip_configured":"10.0.0.5"}]`}, nil
}
