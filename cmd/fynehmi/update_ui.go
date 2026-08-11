package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"modbustcpipserver/internal/updater"
)

var appVersion = "dev"

func updateMessage(current string, release *updater.Release, err error) string {
	if err != nil {
		return "Could not check for updates: " + err.Error()
	}
	if release == nil {
		return "Modbus TCP Simulator " + current + " is up to date."
	}
	name := strings.TrimSpace(release.Name)
	if name == "" {
		name = "Modbus TCP Simulator " + release.TagName
	}
	return "Update available: " + release.TagName + "\n\n" + name
}

func checkForUpdatesOnStartup(window fyne.Window, executable string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	service := updater.Service{HTTP: &http.Client{Timeout: 8 * time.Second}}
	release, err := service.Check(ctx, appVersion)
	if err != nil || release == nil {
		return
	}
	fyne.Do(func() { showUpdateOffer(window, executable, release) })
}

func showManualUpdateDialog(window fyne.Window, executable string) {
	status := widget.NewLabel("Checking for updates…")
	status.Wrapping = fyne.TextWrapWord
	updateButton := widget.NewButton("Update", nil)
	updateButton.Disable()
	dialog.ShowCustom("Updates", "Close", container.NewVBox(status, updateButton), window)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		service := updater.Service{HTTP: &http.Client{Timeout: 8 * time.Second}}
		release, err := service.Check(ctx, appVersion)
		message := updateMessage(appVersion, release, err)
		fyne.Do(func() {
			status.SetText(message)
			if release != nil {
				updateButton.OnTapped = func() { installUpdate(window, executable, release) }
				updateButton.Enable()
			}
		})
	}()
}

func showUpdateOffer(window fyne.Window, executable string, release *updater.Release) {
	message := fmt.Sprintf("A newer Modbus TCP Simulator version is available: %s\n\nThe executable will be downloaded and checksum-verified before installation.", release.TagName)
	dialog.ShowConfirm("Update available", message, func(confirmed bool) {
		if confirmed {
			installUpdate(window, executable, release)
		}
	}, window)
}

func installUpdate(window fyne.Window, executable string, release *updater.Release) {
	progress := dialog.NewCustomWithoutButtons("Installing update", widget.NewProgressBarInfinite(), window)
	progress.Show()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		service := updater.Service{HTTP: &http.Client{Timeout: 2 * time.Minute}}
		source, err := service.Download(ctx, release, filepath.Dir(executable))
		if err == nil {
			err = updater.Start(source, executable)
		}
		fyne.Do(func() {
			progress.Hide()
			if err != nil {
				dialog.ShowError(fmt.Errorf("update failed: %w", err), window)
				return
			}
			window.Close()
			os.Exit(0)
		})
	}()
}
