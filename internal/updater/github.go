package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const GitHubLatestReleaseURL = "https://api.github.com/repos/rjboer/ModbusTCPSimulator/releases/latest"

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

type Service struct {
	HTTP             *http.Client
	LatestReleaseURL string
}

func (service Service) Check(ctx context.Context, current string) (*Release, error) {
	endpoint := service.LatestReleaseURL
	if endpoint == "" {
		endpoint = GitHubLatestReleaseURL
	}
	client := service.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "ModbusTCPSimulator-Updater")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub release check returned %s", response.Status)
	}
	var release Release
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.Draft || release.Prerelease || !newerRelease(current, release.TagName) {
		return nil, nil
	}
	return &release, nil
}

func (service Service) Download(ctx context.Context, release *Release, directory string) (string, error) {
	if release == nil {
		return "", errors.New("release is required")
	}
	var executableAsset, checksumAsset *Asset
	for index := range release.Assets {
		asset := &release.Assets[index]
		switch strings.ToLower(asset.Name) {
		case "modbus-tcp-simulator.exe":
			executableAsset = asset
		case "sha256sums.txt":
			checksumAsset = asset
		}
	}
	if executableAsset == nil || checksumAsset == nil {
		return "", errors.New("release requires modbus-tcp-simulator.exe and SHA256SUMS.txt")
	}
	client := service.HTTP
	if client == nil {
		client = http.DefaultClient
	}

	checksumRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumAsset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	checksumRequest.Header.Set("User-Agent", "ModbusTCPSimulator-Updater")
	checksumResponse, err := client.Do(checksumRequest)
	if err != nil {
		return "", err
	}
	defer checksumResponse.Body.Close()
	if checksumResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned %s", checksumResponse.Status)
	}
	checksumContents, err := io.ReadAll(checksumResponse.Body)
	if err != nil {
		return "", err
	}
	expected, ok := parseChecksum(string(checksumContents), executableAsset.Name)
	if !ok {
		return "", errors.New("SHA256SUMS.txt does not contain the simulator checksum")
	}

	executableRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, executableAsset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	executableRequest.Header.Set("User-Agent", "ModbusTCPSimulator-Updater")
	executableResponse, err := client.Do(executableRequest)
	if err != nil {
		return "", err
	}
	defer executableResponse.Body.Close()
	if executableResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("executable download returned %s", executableResponse.Status)
	}
	temporary, err := os.CreateTemp(directory, ".modbus-tcp-simulator-update-*.exe")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	if _, err := io.Copy(temporary, executableResponse.Body); err != nil {
		temporary.Close()
		os.Remove(temporaryPath)
		return "", err
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporaryPath)
		return "", err
	}
	data, err := os.ReadFile(temporaryPath)
	if err != nil {
		os.Remove(temporaryPath)
		return "", err
	}
	actualBytes := sha256.Sum256(data)
	if actual := hex.EncodeToString(actualBytes[:]); !strings.EqualFold(actual, expected) {
		os.Remove(temporaryPath)
		return "", fmt.Errorf("download checksum mismatch: got %s, want %s", actual, expected)
	}
	return temporaryPath, nil
}
