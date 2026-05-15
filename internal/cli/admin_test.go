package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestAdminMacHostsRequiresForceForAllocate(t *testing.T) {
	app := App{Stdout: io.Discard, Stderr: io.Discard}
	err := app.adminMacHosts(context.Background(), []string{"allocate", "--availability-zone", "eu-west-1a"})
	if err == nil || !strings.Contains(err.Error(), "requires --force") {
		t.Fatalf("err=%v, want force requirement", err)
	}
}

func TestAdminMacHostsRequiresForceForRelease(t *testing.T) {
	app := App{Stdout: io.Discard, Stderr: io.Discard}
	err := app.adminMacHosts(context.Background(), []string{"release", "h-000000000001"})
	if err == nil || !strings.Contains(err.Error(), "requires --force") {
		t.Fatalf("err=%v, want force requirement", err)
	}
}

func TestAdminMacHostsRejectsMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	app := App{Stdout: &stdout, Stderr: io.Discard}
	err := app.adminMacHosts(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "usage: crabbox admin mac-hosts") {
		t.Fatalf("err=%v, want usage error", err)
	}
}

func TestSummarizeMacHostDryRunMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "dry run",
			message: "<Error><Code>DryRunOperation</Code><Message>Request would have succeeded</Message></Error>",
			want:    "DryRunOperation: request would have succeeded",
		},
		{
			name:    "unauthorized",
			message: "<Error><Code>UnauthorizedOperation</Code><Message>provider authorization details omitted</Message></Error>",
			want:    "UnauthorizedOperation: coordinator AWS identity needs EC2 Mac host lifecycle permissions, including ec2:AllocateHosts and ec2:CreateTags",
		},
		{
			name:    "other aws code",
			message: "<Error><Code>HostLimitExceeded</Code><Message>limit exceeded</Message></Error>",
			want:    "HostLimitExceeded",
		},
		{
			name:    "blank",
			message: "",
			want:    "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizeMacHostDryRunMessage(tt.message); got != tt.want {
				t.Fatalf("summary=%q, want %q", got, tt.want)
			}
		})
	}
}
