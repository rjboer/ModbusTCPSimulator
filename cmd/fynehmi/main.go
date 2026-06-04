package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	app2 "modbustcpipserver/internal/app"
	"modbustcpipserver/internal/modbus"
)

func main() {
	projectRoot, err := app2.DetectProjectRoot()
	if err != nil {
		panic(err)
	}
	configRoot := app2.ConfigRoot(projectRoot)

	runtime := app2.NewRuntime(configRoot)
	discovery, err := runtime.Discover()
	if err != nil {
		panic(err)
	}

	ui := app.NewWithID("rjboer.modbus-tcp-simulator.hmi")
	window := ui.NewWindow("Modbus TCP Simulator HMI")
	window.Resize(fyne.NewSize(1400, 900))

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

	registerTable := widget.NewTable(
		func() (int, int) {
			return len(registerRows) + 1, 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapOff

			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.SetText(registerTableHeader(id.Col))
				return
			}

			label.TextStyle = fyne.TextStyle{}
			row := registerRows[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(row.Block)
			case 1:
				label.SetText(row.Address)
			case 2:
				label.SetText(row.Hex)
			case 3:
				label.SetText(row.Value)
			case 4:
				label.SetText(row.Description)
			}
		},
	)
	registerTable.SetColumnWidth(0, 140)
	registerTable.SetColumnWidth(1, 90)
	registerTable.SetColumnWidth(2, 90)
	registerTable.SetColumnWidth(3, 120)
	registerTable.SetColumnWidth(4, 420)
	registerTable.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 || id.Col != 3 || id.Row-1 >= len(registerRows) {
			return
		}

		row := registerRows[id.Row-1]
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
		path := modbus.DefaultLogExportPath()
		if err := runtime.ExportLogs(path); err != nil {
			dialog.ShowError(err, window)
			return
		}
		refreshStatus(fmt.Sprintf("Log exported to %s", path))
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

	trafficCard := widget.NewCard(
		"Traffic Overview",
		"Connection counters and network load",
		container.NewBorder(
			nil,
			nil,
			nil,
			nil,
			container.NewVSplit(container.NewScroll(trafficSummary), connectionList),
		),
	)

	centerTop := container.NewVBox(
		buttonBar,
		configDetailsCard,
		trafficCard,
	)

	registerCard := widget.NewCard(
		"Register Map",
		"Live register values and descriptions",
		registerTable,
	)

	upperSplit := container.NewHSplit(centerTop, registerCard)
	upperSplit.Offset = 0.42

	logsCard := widget.NewCard(
		"Logs",
		"Live server log including connection information",
		container.NewScroll(logView),
	)

	rightDashboard := container.NewVSplit(upperSplit, logsCard)
	rightDashboard.Offset = 0.62

	dashboardSplit := container.NewHSplit(leftPane, rightDashboard)
	dashboardSplit.Offset = 0.20

	introductionText := widget.NewMultiLineEntry()
	introductionText.Disable()
	introductionText.Wrapping = fyne.TextWrapWord
	introductionText.SetText(strings.TrimSpace(`
Modbus TCP Simulator HMI

This screen is intended as a single operator dashboard for the simulator.

- Select a configuration on the left.
- Start the selected mock server with the Start button.
- Review network settings and profile details in the center.
- Inspect the live register map on the right.
- Watch logs and connection activity at the bottom.
- Traffic overview shows bandwidth, message rate, and active client details.

The HMI is designed for engineering work: testing PLC communication, validating register mappings, and debugging client behavior without physical hardware.
`))

	introductionCard := widget.NewCard(
		"Introduction",
		"How to use the simulator dashboard",
		introductionText,
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
			widget.NewLabel(fmt.Sprintf("Project root: %s", projectRoot)),
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
