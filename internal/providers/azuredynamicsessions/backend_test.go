package azuredynamicsessions

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestResolveSessionIDRequiresLocalClaim(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	backend := &azureDynamicSessionsBackend{}
	client := &recordingAzureDynamicSessionsAPI{}

	_, _, err := backend.resolveSessionID(context.Background(), client, "azds-external", t.TempDir(), false)
	if err == nil || !strings.Contains(err.Error(), "not claimed by Crabbox") {
		t.Fatalf("resolve unclaimed session err=%v, want claim boundary error", err)
	}
	if client.getSessionCalls != 0 {
		t.Fatalf("GetSession calls = %d, want 0", client.getSessionCalls)
	}
}

func TestResolveSessionIDUsesClaimedSlug(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repoA := t.TempDir()
	repoB := t.TempDir()
	if err := claimLeaseForRepoProvider("azds-claimed", "claimed-session", providerName, repoA, time.Minute, false); err != nil {
		t.Fatal(err)
	}
	backend := &azureDynamicSessionsBackend{}
	client := &recordingAzureDynamicSessionsAPI{}

	if _, _, err := backend.resolveSessionID(context.Background(), client, "claimed-session", repoB, false); err == nil || !strings.Contains(err.Error(), "use --reclaim") {
		t.Fatalf("resolve without reclaim err=%v, want reclaim guard", err)
	}
	leaseID, slug, err := backend.resolveSessionID(context.Background(), client, "claimed-session", repoB, true)
	if err != nil {
		t.Fatal(err)
	}
	if leaseID != "azds-claimed" || slug != "claimed-session" {
		t.Fatalf("resolved lease=%q slug=%q", leaseID, slug)
	}
	if client.getSessionCalls != 0 {
		t.Fatalf("GetSession calls = %d, want 0", client.getSessionCalls)
	}
}

type recordingAzureDynamicSessionsAPI struct {
	getSessionCalls int
}

func (r *recordingAzureDynamicSessionsAPI) CheckRunner(context.Context, string) error {
	return nil
}

func (r *recordingAzureDynamicSessionsAPI) UploadFile(context.Context, string, string, string) error {
	return nil
}

func (r *recordingAzureDynamicSessionsAPI) ExecStream(context.Context, string, azureDynamicSessionsExecRequest, io.Writer, io.Writer) (int, error) {
	return 0, nil
}

func (r *recordingAzureDynamicSessionsAPI) GetSession(context.Context, string) (azureDynamicSessionsSession, error) {
	r.getSessionCalls++
	return azureDynamicSessionsSession{}, nil
}

func (r *recordingAzureDynamicSessionsAPI) ListSessions(context.Context) ([]azureDynamicSessionsSession, error) {
	return nil, nil
}

func (r *recordingAzureDynamicSessionsAPI) DeleteSession(context.Context, string) error {
	return nil
}
