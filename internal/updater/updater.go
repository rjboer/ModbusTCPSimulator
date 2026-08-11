package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func parseVersion(tag string) ([3]int, bool) {
	var version [3]int
	value := strings.TrimSpace(tag)
	if !strings.HasPrefix(value, "v") {
		return version, false
	}
	parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(parts) != len(version) {
		return version, false
	}
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return version, false
		}
		version[index] = parsed
	}
	return version, true
}

func newerRelease(current, latest string) bool {
	latestVersion, latestOK := parseVersion(latest)
	if !latestOK {
		return false
	}
	currentVersion, currentOK := parseVersion(current)
	if !currentOK {
		return true
	}
	for index := range latestVersion {
		if latestVersion[index] != currentVersion[index] {
			return latestVersion[index] > currentVersion[index]
		}
	}
	return false
}

func parseChecksum(contents, filename string) (string, bool) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(strings.TrimPrefix(fields[1], "*"), filename) {
			continue
		}
		hash := strings.ToLower(fields[0])
		if len(hash) != sha256.Size*2 {
			continue
		}
		if _, err := hex.DecodeString(hash); err != nil {
			continue
		}
		return hash, true
	}
	return "", false
}

func Apply(source, target string) error {
	backup := target + ".previous"
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous executable backup: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt < 80; attempt++ {
		if err := os.Rename(target, backup); err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if err := os.Rename(source, target); err != nil {
			_ = os.Rename(backup, target)
			return fmt.Errorf("install updated executable: %w", err)
		}
		return nil
	}
	return fmt.Errorf("could not replace running executable: %w", lastErr)
}
