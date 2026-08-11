package main

import (
	"errors"
	"testing"

	"modbustcpipserver/internal/updater"
)

func TestUpdateMessageReportsCurrentVersion(t *testing.T) {
	got := updateMessage("v1.2.3", nil, nil)
	if got != "Modbus TCP Simulator v1.2.3 is up to date." {
		t.Fatalf("updateMessage() = %q", got)
	}
}

func TestUpdateMessageReportsAvailableRelease(t *testing.T) {
	release := &updater.Release{TagName: "v1.3.0", Name: "Modbus TCP Simulator 1.3.0"}
	got := updateMessage("v1.2.3", release, nil)
	if got != "Update available: v1.3.0\n\nModbus TCP Simulator 1.3.0" {
		t.Fatalf("updateMessage() = %q", got)
	}
}

func TestUpdateMessageReportsError(t *testing.T) {
	got := updateMessage("v1.2.3", nil, errors.New("network unavailable"))
	if got != "Could not check for updates: network unavailable" {
		t.Fatalf("updateMessage() = %q", got)
	}
}
