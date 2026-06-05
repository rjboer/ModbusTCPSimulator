package main

import (
	"fmt"
	"image/color"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	app2 "modbustcpipserver/internal/app"
	"modbustcpipserver/internal/modbus"
)

func main() {
	const (
		preferenceLaunchCount         = "launch_count"
		preferenceLinkedInPromptCount = "linkedin_prompt_count"
	)

	appRoot, err := app2.DetectProjectRoot()
	if err != nil {
		panic(err)
	}
	configRoot := app2.ConfigRoot(appRoot)

	runtime := app2.NewRuntime(configRoot)
	discovery, err := runtime.Discover()
	if err != nil {
		panic(err)
	}

	ui := app.NewWithID("rjboer.modbus-tcp-simulator.hmi")
	window := ui.NewWindow("Modbus TCP Simulator HMI")
	window.Resize(fyne.NewSize(1400, 900))

	prefs := ui.Preferences()
	launchCount := prefs.IntWithFallback(preferenceLaunchCount, 0) + 1
	prefs.SetInt(preferenceLaunchCount, launchCount)

	done := make(chan struct{})
	var stopRefreshOnce sync.Once
	stopRefreshLoop := func() {
		stopRefreshOnce.Do(func() {
			close(done)
		})
	}

	statusLabel := widget.NewLabel("Ready")
	configSummary := widget.NewTextGrid()
	trafficSummary := widget.NewTextGrid()
	logView := widget.NewTextGrid()

	connectionRows := []string{"No active connections"}
	connectionList := widget.NewList(
		func() int { return len(connectionRows) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(connectionRows[id])
		},
	)

	var selectedIndex int
	validConfigs := discovery.Valid
	configRows := buildConfigRows(validConfigs)

	configList := widget.NewList(
		func() int { return len(configRows) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(configRows[id])
		},
	)

	startButton := widget.NewButton("Start Server", nil)
	stopButton := widget.NewButton("Stop Server", nil)
	registerRows := []registerRow{}
	var refreshStatus func(string)
	var refreshTelemetry func()
	const (
		registerBlockColWidth       float32 = 140
		registerAddressColWidth     float32 = 90
		registerHexColWidth         float32 = 90
		registerValueColWidth       float32 = 120
		registerDescriptionColWidth float32 = 420
	)

	registerTable := widget.NewTable(
		func() (int, int) {
			return len(registerRows), 5
		},
		func() fyne.CanvasObject {
			text := canvas.NewText("", theme.Color(theme.ColorNameForeground))
			text.TextSize = theme.TextSize()
			return text
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			text := obj.(*canvas.Text)
			text.Alignment = fyne.TextAlignLeading
			text.TextSize = theme.TextSize()
			text.Color = theme.Color(theme.ColorNameForeground)
			text.TextStyle = fyne.TextStyle{}
			row := registerRows[id.Row]
			switch id.Col {
			case 0:
				text.Text = row.Block
			case 1:
				text.Text = row.Address
			case 2:
				text.Text = row.Hex
			case 3:
				text.Text = row.Value
				text.Color = registerValueColor(row)
			case 4:
				text.Text = row.Description
			}
			text.Refresh()
		},
	)
	registerTable.SetColumnWidth(0, registerBlockColWidth)
	registerTable.SetColumnWidth(1, registerAddressColWidth)
	registerTable.SetColumnWidth(2, registerHexColWidth)
	registerTable.SetColumnWidth(3, registerValueColWidth)
	registerTable.SetColumnWidth(4, registerDescriptionColWidth)
	registerTable.OnSelected = func(id widget.TableCellID) {
		if id.Col != 3 || id.Row >= len(registerRows) {
			return
		}

		row := registerRows[id.Row]
		if row.IsBool {
			nextValue := row.Value != "true"
			if err := runtime.SetBoolValue(row.Block, row.AddressNum, nextValue); err != nil {
				dialog.ShowError(err, window)
				registerTable.Unselect(id)
				return
			}
			refreshTelemetry()
			refreshStatus(fmt.Sprintf("%s %s set to %t", row.Block, row.Address, nextValue))
			registerTable.Unselect(id)
			return
		}

		registerTable.Unselect(id)

		entry := widget.NewEntry()
		entry.SetText(strings.Fields(row.Value)[0])
		var editor *dialog.FormDialog
		applyValue := func() {
			value, err := strconv.ParseUint(strings.TrimSpace(entry.Text), 0, 16)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid value %q", entry.Text), window)
				return
			}
			if err := runtime.SetRegisterValue(row.Block, row.AddressNum, uint16(value)); err != nil {
				dialog.ShowError(err, window)
				return
			}
			if editor != nil {
				editor.Hide()
			}
			refreshTelemetry()
			refreshStatus(fmt.Sprintf("%s %s set to %d", row.Block, row.Address, value))
		}

		entry.OnSubmitted = func(string) {
			applyValue()
		}

		previousKeyHandler := window.Canvas().OnTypedKey()
		editor = dialog.NewForm(
			fmt.Sprintf("Edit %s %s", row.Block, row.Address),
			"Apply",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Value", entry),
			},
			func(confirmed bool) {
				if !confirmed {
					return
				}
				applyValue()
			},
			window,
		)
		editor.SetOnClosed(func() {
			window.Canvas().SetOnTypedKey(previousKeyHandler)
		})
		window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name == fyne.KeyEscape {
				editor.Hide()
				return
			}
			if previousKeyHandler != nil {
				previousKeyHandler(ev)
			}
		})
		editor.Show()
		window.Canvas().Focus(entry)
	}

	updateSelectedConfig := func() {
		if len(validConfigs) == 0 {
			configSummary.SetText("No valid configs found.")
			registerRows = nil
			registerTable.Refresh()
			return
		}

		if selectedIndex >= len(validConfigs) {
			selectedIndex = 0
		}
		selected := validConfigs[selectedIndex]
		_ = runtime.SelectConfig(selected.Path)
		configSummary.SetText(formatConfigSummary(selected))
	}

	refreshStatus = func(message string) {
		statusLabel.SetText(message)
	}

	updateServerControls := func(running bool) {
		if running {
			startButton.Importance = widget.SuccessImportance
			startButton.SetText("Server Running")
			startButton.Disable()
			stopButton.Importance = widget.WarningImportance
			stopButton.Enable()
		} else {
			startButton.Importance = widget.MediumImportance
			startButton.SetText("Start Server")
			startButton.Enable()
			stopButton.Importance = widget.MediumImportance
			stopButton.Disable()
		}
		startButton.Refresh()
		stopButton.Refresh()
	}

	refreshTelemetry = func() {
		running, runtimeErr := runtime.SyncState()
		if runtimeErr != nil {
			refreshStatus(fmt.Sprintf("Server error: %v", runtimeErr))
			dialog.ShowError(runtimeErr, window)
		}
		updateServerControls(running)

		lines, _ := runtime.LogsSince(0)
		logView.SetText(strings.Join(lines, "\n"))

		snapshot := runtime.TrafficSnapshot()
		trafficSummary.SetText(formatTrafficSummary(snapshot))

		connectionRows = buildConnectionRows(snapshot.Connections)
		connectionList.Refresh()

		_, dataSnapshot := runtime.SnapshotData()
		registerRows = buildRegisterRows(dataSnapshot)
		registerTable.Refresh()
	}

	configList.OnSelected = func(id widget.ListItemID) {
		selectedIndex = id
		updateSelectedConfig()
		refreshStatus(fmt.Sprintf("Selected %s", validConfigs[id].RelPath))
	}

	refreshButton := widget.NewButton("Refresh Configs", func() {
		discovery, err := runtime.Discover()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		validConfigs = discovery.Valid
		configRows = buildConfigRows(validConfigs)
		configList.Refresh()
		updateSelectedConfig()
		refreshStatus(fmt.Sprintf("Discovered %d valid configs", len(validConfigs)))
	})

	startButton.OnTapped = func() {
		if err := runtime.Start(); err != nil {
			dialog.ShowError(err, window)
			refreshStatus(fmt.Sprintf("Start failed: %v", err))
			updateServerControls(false)
			return
		}
		refreshTelemetry()
		refreshStatus("Server running")
	}

	stopButton.OnTapped = func() {
		if err := runtime.Stop(); err != nil {
			dialog.ShowError(err, window)
			refreshStatus(fmt.Sprintf("Stop failed: %v", err))
			return
		}
		refreshTelemetry()
		refreshStatus("Server stopped")
	}

	exportButton := widget.NewButton("Export Log", func() {
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if writer == nil {
				return
			}
			defer writer.Close()

			lines, _ := runtime.LogsSince(0)
			if _, err := io.WriteString(writer, strings.Join(lines, "\n")); err != nil {
				dialog.ShowError(err, window)
				return
			}
			refreshStatus(fmt.Sprintf("Log exported to %s", writer.URI().Path()))
		}, window)
		saveDialog.SetFileName("modbus-simulator.log")
		saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".log", ".txt"}))
		saveDialog.Show()
	})

	leftPane := container.NewPadded(container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Discovered Configurations"),
			refreshButton,
		),
		nil,
		nil,
		nil,
		configList,
	))

	buttonBar := container.NewGridWithColumns(3, startButton, stopButton, exportButton)

	configDetailsCard := widget.NewCard(
		"Config Details",
		"Selected profile and network settings",
		container.NewScroll(configSummary),
	)

	trafficSplit := container.NewVSplit(container.NewScroll(trafficSummary), connectionList)
	trafficSplit.Offset = 0.58

	trafficCard := widget.NewCard(
		"Traffic Overview",
		"Connection counters and network load",
		trafficSplit,
	)

	centerColumn := container.NewVSplit(
		container.NewPadded(buttonBar),
		trafficCard,
	)
	centerColumn.Offset = 0.12

	makeRegisterHeaderCell := func(title string, width float32) fyne.CanvasObject {
		label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		label.Wrapping = fyne.TextWrapOff
		return container.New(layout.NewGridWrapLayout(fyne.NewSize(width, label.MinSize().Height+12)), container.NewPadded(label))
	}

	registerHeader := container.NewHBox(
		makeRegisterHeaderCell(registerTableHeader(0), registerBlockColWidth),
		makeRegisterHeaderCell(registerTableHeader(1), registerAddressColWidth),
		makeRegisterHeaderCell(registerTableHeader(2), registerHexColWidth),
		makeRegisterHeaderCell(registerTableHeader(3), registerValueColWidth),
		makeRegisterHeaderCell(registerTableHeader(4), registerDescriptionColWidth),
	)

	registerCard := widget.NewCard(
		"Register Map",
		"Live register values and descriptions",
		container.NewBorder(registerHeader, nil, nil, nil, registerTable),
	)

	leftColumn := container.NewVSplit(leftPane, configDetailsCard)
	leftColumn.Offset = 0.56

	upperSplit := container.NewHSplit(centerColumn, registerCard)
	upperSplit.Offset = 0.37

	logsCard := widget.NewCard(
		"Logs",
		"Live server log including connection information",
		container.NewScroll(logView),
	)

	rightDashboard := container.NewVSplit(upperSplit, logsCard)
	rightDashboard.Offset = 0.72

	dashboardSplit := container.NewHSplit(leftColumn, rightDashboard)
	dashboardSplit.Offset = 0.24

	makeInfoHeading := func(text string) *widget.Label {
		label := widget.NewLabel(text)
		label.Wrapping = fyne.TextWrapWord
		label.TextStyle = fyne.TextStyle{Bold: true}
		return label
	}

	makeInfoBody := func(text string) *widget.Label {
		label := widget.NewLabel(text)
		label.Wrapping = fyne.TextWrapWord
		return label
	}

	linkedinURL, _ := url.Parse("https://www.linkedin.com/in/rjboer")
	linkedinLink := widget.NewHyperlink("https://www.linkedin.com/in/rjboer", linkedinURL)

	introductionContent := container.NewVBox(
		makeInfoHeading("Modbus TCP Simulator HMI"),
		makeInfoBody("Please add me on LinkedIn. I do not ask payment for this software."),
		makeInfoBody("This dashboard gives you one place to start a simulator, inspect runtime behavior, and diagnose Modbus TCP communication without needing the physical hardware on your desk."),
		widget.NewSeparator(),
		makeInfoHeading("What You Can Do Here"),
		makeInfoBody("- Select a configuration.\n- Start or stop a simulated Modbus TCP server.\n- Inspect and edit live register values.\n- Monitor logs, active connections, and traffic rates."),
		widget.NewSeparator(),
		makeInfoHeading("Why This Is Useful"),
		makeInfoBody("Use it for PLC development, register-map validation, unit-ID checks, traffic inspection, and debugging client polling behavior before real hardware is available."),
		widget.NewSeparator(),
		makeInfoHeading("Dashboard Areas"),
		makeInfoBody("Configurations: choose the mock profile.\nConfig Details: review the selected profile.\nRegister Map: inspect and change live values.\nTraffic Overview: view throughput, message rate, and active connections.\nLogs: inspect requests, warnings, and shutdown activity."),
		widget.NewSeparator(),
		makeInfoHeading("Support"),
		makeInfoBody("Please add me on LinkedIn."),
		makeInfoBody("Roelof Jan Boer"),
		linkedinLink,
	)

	introductionCard := widget.NewCard(
		"Introduction",
		"How to use the simulator dashboard",
		container.NewScroll(introductionContent),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Introduction", introductionCard),
		container.NewTabItem("Dashboard", dashboardSplit),
	)

	content := container.NewBorder(
		nil,
		container.NewHBox(
			statusLabel,
			layout.NewSpacer(),
			widget.NewLabel(fmt.Sprintf("App root: %s", appRoot)),
			widget.NewLabel(fmt.Sprintf("Config root: %s", configRoot)),
		),
		nil,
		nil,
		tabs,
	)

	window.SetContent(content)

	updateSelectedConfig()
	updateServerControls(false)
	if len(validConfigs) > 0 {
		configList.Select(0)
	}
	refreshTelemetry()

	window.SetCloseIntercept(func() {
		stopRefreshLoop()
		_ = runtime.Stop()
		window.SetCloseIntercept(nil)
		window.Close()
	})

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fyne.Do(refreshTelemetry)
			}
		}
	}()

	if launchCount <= 2 && prefs.IntWithFallback(preferenceLinkedInPromptCount, 0) < 2 {
		go func() {
			time.Sleep(400 * time.Millisecond)
			fyne.Do(func() {
				var prompt dialog.Dialog
				openButton := widget.NewButton("Open LinkedIn", func() {
					prefs.SetInt(preferenceLinkedInPromptCount, prefs.IntWithFallback(preferenceLinkedInPromptCount, 0)+1)
					prompt.Hide()
					if err := ui.OpenURL(linkedinURL); err != nil {
						dialog.ShowError(err, window)
					}
				})
				openButton.Importance = widget.HighImportance
				promptContent := container.NewVBox(
					widget.NewLabel("I don't ask for payment. I just ask that you add me on LinkedIn."),
					openButton,
				)
				prompt = dialog.NewCustomWithoutButtons(
					"Support The Project",
					promptContent,
					window,
				)
				prompt.Show()
			})
		}()
	}

	window.ShowAndRun()
	stopRefreshLoop()
	_ = runtime.Stop()
}

func buildConfigRows(items []modbus.DiscoveredConfig) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | %s | %s:%d", item.Config.Name, item.Config.DeviceType, item.Config.ListenAddress, item.Config.Port))
	}
	return rows
}

func registerValueColor(row registerRow) color.Color {
	if row.IsBool {
		if row.Value == "true" {
			return color.NRGBA{R: 140, G: 210, B: 160, A: 255}
		}
		return color.NRGBA{R: 210, G: 140, B: 140, A: 255}
	}
	valueText := strings.TrimSpace(strings.Fields(row.Value)[0])
	if value, err := strconv.ParseUint(valueText, 0, 64); err == nil && value == 0 {
		return theme.Color(theme.ColorNameForeground)
	}
	return color.NRGBA{R: 145, G: 185, B: 235, A: 255}
}

type registerRow struct {
	Block       string
	AddressNum  uint16
	Address     string
	Hex         string
	Value       string
	Description string
	IsBool      bool
}

func registerTableHeader(col int) string {
	switch col {
	case 0:
		return "Block"
	case 1:
		return "Address"
	case 2:
		return "Hex"
	case 3:
		return "Value"
	case 4:
		return "Description"
	default:
		return ""
	}
}

func formatConfigSummary(item modbus.DiscoveredConfig) string {
	return fmt.Sprintf(
		"Path            %s\nName            %s\nDevice type     %s\nProfile         %s\nEndpoint        %s:%d\nUnit IDs        %v\nProtocol        %s\nData bits       %d\nParity          %s\nStop bits       %d\nMax clients     %d\nIdle timeout    %d ms",
		item.RelPath,
		item.Config.Name,
		item.Config.DeviceType,
		item.Config.DeviceProfile,
		item.Config.ListenAddress,
		item.Config.Port,
		item.Config.UnitIDs,
		item.Config.Network.Protocol,
		item.Config.Network.DataBits,
		item.Config.Network.Parity,
		item.Config.Network.StopBits,
		item.Config.Connection.MaxClients,
		item.Config.Connection.IdleTimeoutMs,
	)
}

func formatTrafficSummary(snapshot modbus.TrafficSnapshot) string {
	if snapshot.StartedAt.IsZero() {
		return "Server not running."
	}

	return fmt.Sprintf(
		"Started           %s\nActive clients    %d\nInbound           %s/s\nOutbound          %s/s\nTotal load        %s/s\nMessages          %.2f msg/s\nTotal bytes in    %s\nTotal bytes out   %s\nTotal messages in %d\nTotal messages out %d",
		snapshot.StartedAt.Format(time.RFC3339),
		snapshot.ActiveClients,
		formatBytesPerSecond(snapshot.BytesInPerSecond),
		formatBytesPerSecond(snapshot.BytesOutPerSecond),
		formatBytesPerSecond(snapshot.BytesInPerSecond+snapshot.BytesOutPerSecond),
		snapshot.MessagesPerSecond,
		formatBytes(float64(snapshot.TotalBytesIn)),
		formatBytes(float64(snapshot.TotalBytesOut)),
		snapshot.TotalMessagesIn,
		snapshot.TotalMessagesOut,
	)
}

func buildConnectionRows(connections []modbus.ConnectionStats) []string {
	if len(connections) == 0 {
		return []string{"No active connections"}
	}

	rows := make([]string, 0, len(connections))
	for _, conn := range connections {
		rows = append(rows, fmt.Sprintf(
			"%s | connected %s | in=%s out=%s | msg in=%d out=%d | last unit=%d func=0x%02X",
			conn.RemoteAddr,
			conn.ConnectedAt.Format(time.Kitchen),
			formatBytes(float64(conn.BytesIn)),
			formatBytes(float64(conn.BytesOut)),
			conn.MessagesIn,
			conn.MessagesOut,
			conn.LastUnitID,
			conn.LastFunction,
		))
	}
	return rows
}

func formatBytesPerSecond(value float64) string {
	return formatBytes(value)
}

func formatBytes(value float64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
	)

	switch {
	case value >= mb:
		return fmt.Sprintf("%.2f MB", value/mb)
	case value >= kb:
		return fmt.Sprintf("%.2f KB", value/kb)
	default:
		return fmt.Sprintf("%.0f B", value)
	}
}

func buildRegisterRows(snapshot modbus.DataStoreSnapshot) []registerRow {
	rows := make([]registerRow, 0,
		len(snapshot.Coils.Values)+
			len(snapshot.DiscreteInputs.Values)+
			len(snapshot.Holding.Values)+
			len(snapshot.Input.Values),
	)

	rows = appendBoolRows(rows, "Coils", snapshot.Coils)
	rows = appendBoolRows(rows, "Discrete Inputs", snapshot.DiscreteInputs)
	rows = appendRegisterRows(rows, "Holding Registers", snapshot.Holding)
	rows = appendRegisterRows(rows, "Input Registers", snapshot.Input)

	return rows
}

func appendBoolRows(rows []registerRow, blockName string, block modbus.BoolBlock) []registerRow {
	for i, value := range block.Values {
		address := block.StartAddress + uint16(i)
		rows = append(rows, registerRow{
			Block:       blockName,
			AddressNum:  address,
			Address:     strconv.Itoa(int(address)),
			Hex:         fmt.Sprintf("0x%04X", address),
			Value:       strconv.FormatBool(value),
			Description: descriptionAt(block.Descriptions, i),
			IsBool:      true,
		})
	}
	return rows
}

func appendRegisterRows(rows []registerRow, blockName string, block modbus.RegisterBlock) []registerRow {
	for i, value := range block.Values {
		address := block.StartAddress + uint16(i)
		rows = append(rows, registerRow{
			Block:       blockName,
			AddressNum:  address,
			Address:     strconv.Itoa(int(address)),
			Hex:         fmt.Sprintf("0x%04X", address),
			Value:       fmt.Sprintf("%d (0x%04X)", value, value),
			Description: descriptionAt(block.Descriptions, i),
		})
	}
	return rows
}

func descriptionAt(descriptions []string, index int) string {
	if index < 0 || index >= len(descriptions) {
		return ""
	}
	return descriptions[index]
}
