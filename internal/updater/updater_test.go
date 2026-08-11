package updater

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersionAcceptsStableReleaseTags(t *testing.T) {
	got, ok := parseVersion("v2.14.3")
	if !ok || got != [3]int{2, 14, 3} {
		t.Fatalf("parseVersion() = %#v, %v", got, ok)
	}
	if _, ok := parseVersion("v2.14.3-beta"); ok {
		t.Fatal("prerelease tag unexpectedly accepted")
	}
}

func TestServiceCheckReturnsNewStableRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"tag_name":"v1.4.0","name":"Version 1.4.0","draft":false,"prerelease":false}`)
	}))
	defer server.Close()

	service := Service{HTTP: server.Client(), LatestReleaseURL: server.URL}
	release, err := service.Check(context.Background(), "v1.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if release == nil || release.TagName != "v1.4.0" {
		t.Fatalf("release = %#v", release)
	}
}

func TestServiceDownloadVerifiesSimulatorChecksum(t *testing.T) {
	executable := []byte("verified simulator executable")
	hash := sha256.Sum256(executable)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/modbus-tcp-simulator.exe":
			_, _ = writer.Write(executable)
		case "/SHA256SUMS.txt":
			fmt.Fprintf(writer, "%x  modbus-tcp-simulator.exe\n", hash)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	release := &Release{Assets: []Asset{
		{Name: "modbus-tcp-simulator.exe", BrowserDownloadURL: server.URL + "/modbus-tcp-simulator.exe"},
		{Name: "SHA256SUMS.txt", BrowserDownloadURL: server.URL + "/SHA256SUMS.txt"},
	}}
	service := Service{HTTP: server.Client()}
	path, err := service.Download(context.Background(), release, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(executable) {
		t.Fatalf("downloaded executable = %q", data)
	}
}

func TestServiceDownloadRejectsChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/modbus-tcp-simulator.exe":
			fmt.Fprint(writer, "tampered executable")
		case "/SHA256SUMS.txt":
			fmt.Fprint(writer, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  modbus-tcp-simulator.exe\n")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	release := &Release{Assets: []Asset{
		{Name: "modbus-tcp-simulator.exe", BrowserDownloadURL: server.URL + "/modbus-tcp-simulator.exe"},
		{Name: "SHA256SUMS.txt", BrowserDownloadURL: server.URL + "/SHA256SUMS.txt"},
	}}
	service := Service{HTTP: server.Client()}
	path, err := service.Download(context.Background(), release, t.TempDir())
	if err == nil {
		if path != "" {
			_ = os.Remove(path)
		}
		t.Fatal("checksum mismatch was accepted")
	}
}

func TestNewerReleaseComparesStableVersions(t *testing.T) {
	if !newerRelease("v1.2.3", "v1.3.0") {
		t.Fatal("newer minor release was not detected")
	}
	if newerRelease("v1.3.0", "v1.2.9") {
		t.Fatal("older release was detected as newer")
	}
	if !newerRelease("dev", "v1.0.0") {
		t.Fatal("stable release was not offered to a development build")
	}
}

func TestParseChecksumSelectsSimulatorExecutable(t *testing.T) {
	contents := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.exe\n" +
		"ABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD  modbus-tcp-simulator.exe\n"
	got, ok := parseChecksum(contents, "modbus-tcp-simulator.exe")
	if !ok || got != "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("parseChecksum() = %q, %v", got, ok)
	}
}

func TestApplyReplacesExecutableAndKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "update.exe")
	target := filepath.Join(dir, "modbus-tcp-simulator.exe")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(source, target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("updated executable = %q", data)
	}
	backup, err := os.ReadFile(target + ".previous")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "old" {
		t.Fatalf("backup executable = %q", backup)
	}
}

func TestReplacementCommandCarriesSourceAndTarget(t *testing.T) {
	command := replacementCommand("simulator.exe", "download.exe", "installed.exe")
	want := []string{"simulator.exe", "-apply-update=download.exe", "-update-target=installed.exe"}
	if len(command.Args) != len(want) {
		t.Fatalf("command args = %#v", command.Args)
	}
	for index := range want {
		if command.Args[index] != want[index] {
			t.Fatalf("command args = %#v", command.Args)
		}
	}
}
