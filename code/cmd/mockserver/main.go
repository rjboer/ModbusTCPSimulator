package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"modbustcpipserver/internal/modbus"
)

const appTitle = "Modbus TCP Simulator | Roelof Jan Boer"

func main() {
	configPath := flag.String("config", "", "path to a specific mock server configuration")
	flag.Parse()

	logSink := modbus.NewMemoryLogSink()
	logger := log.New(io.MultiWriter(os.Stdout, logSink), "mock-modbus ", log.LstdFlags|log.Lmicroseconds)

	cfg, selectedPath, err := resolveConfig(*configPath, logger)
	if err != nil {
		logger.Fatalf("resolve config: %v", err)
	}
	logger.Printf("selected config: %s", selectedPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := modbus.NewServer(cfg, logger)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe(ctx)
	}()

	menuErr := runRuntimeMenu(ctx, stop, server, logSink, logger, serverErr)
	if menuErr != nil && !errors.Is(menuErr, context.Canceled) {
		if errors.Is(menuErr, io.EOF) {
			stop()
		}
		logger.Fatalf("runtime menu failed: %v", menuErr)
	}

	if errors.Is(menuErr, context.Canceled) {
		if err := <-serverErr; err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("server stopped with error: %v", err)
		}
	}
}

func resolveConfig(configPath string, logger *log.Logger) (modbus.Config, string, error) {
	if strings.TrimSpace(configPath) != "" {
		cfg, renderedConfig, created, err := modbus.LoadOrCreateConfig(configPath)
		if err != nil {
			return modbus.Config{}, "", err
		}
		if created {
			logger.Printf("config %q was not found; created default profile config", configPath)
			logger.Printf("generated config:\n%s", renderedConfig)
		}
		reader := bufio.NewReader(os.Stdin)
		updatedCfg, path, _, err := configureSelectedConfig(reader, modbus.DiscoveredConfig{
			Path:     configPath,
			RelPath:  filepath.Base(configPath),
			Config:   cfg,
			Rendered: renderedConfig,
		})
		if err != nil {
			return modbus.Config{}, "", err
		}
		return updatedCfg, path, nil
	}

	root, err := os.Getwd()
	if err != nil {
		return modbus.Config{}, "", fmt.Errorf("determine scan root: %w", err)
	}

	discovery, err := modbus.DiscoverConfigs(root)
	if err != nil {
		return modbus.Config{}, "", fmt.Errorf("scan configs in %s: %w", root, err)
	}

	if len(discovery.Valid) == 0 {
		defaultPath := filepath.Join(root, "config.default_c.json")
		cfg, renderedConfig, created, err := modbus.LoadOrCreateConfig(defaultPath)
		if err != nil {
			return modbus.Config{}, "", err
		}
		if created {
			logger.Printf("no valid JSON server configs found under %s", root)
			logger.Printf("created default profile config at %q", defaultPath)
			logger.Printf("generated config:\n%s", renderedConfig)
		}

		discovery.Valid = append(discovery.Valid, modbus.DiscoveredConfig{
			Path:     defaultPath,
			RelPath:  "config.default_c.json",
			Config:   cfg,
			Rendered: renderedConfig,
		})
	}

	return chooseDiscoveredConfig(discovery)
}

func chooseDiscoveredConfig(discovery modbus.DiscoveryResult) (modbus.Config, string, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		printMenuHeader()
		fmt.Printf("Config scan root: %s\n", discovery.Root)
		fmt.Printf("Valid server configs: %d\n", len(discovery.Valid))
		if len(discovery.Invalid) > 0 {
			fmt.Printf("Ignored JSON files: %d\n", len(discovery.Invalid))
			for _, item := range discovery.Invalid {
				fmt.Printf("  - %s: %v\n", item.RelPath, item.Err)
			}
		}
		fmt.Println()
		fmt.Println("Choose a server configuration:")
		for i, item := range discovery.Valid {
			fmt.Printf("  %d. %s | %s | %s:%d\n",
				i+1,
				item.Config.Name,
				item.Config.DeviceType,
				item.Config.ListenAddress,
				item.Config.Port,
			)
		}

		fmt.Printf("Enter selection [1-%d] or q to quit: ", len(discovery.Valid))
		line, err := reader.ReadString('\n')
		if err != nil {
			return modbus.Config{}, "", fmt.Errorf("read menu input: %w", err)
		}

		input := strings.TrimSpace(line)
		if strings.EqualFold(input, "q") {
			return modbus.Config{}, "", fmt.Errorf("startup cancelled")
		}

		index, err := strconv.Atoi(input)
		if err != nil || index < 1 || index > len(discovery.Valid) {
			fmt.Println("Invalid selection.")
			continue
		}

		selected := discovery.Valid[index-1]
		cfg, path, back, err := configureSelectedConfig(reader, selected)
		if err != nil {
			return modbus.Config{}, "", err
		}
		if back {
			continue
		}
		discovery.Valid[index-1].Config = cfg
		return cfg, path, nil
	}
}

func configureSelectedConfig(reader *bufio.Reader, selected modbus.DiscoveredConfig) (modbus.Config, string, bool, error) {
	cfg := selected.Config

	for {
		printMenuHeader()
		fmt.Printf("Selected config: %s\n", selected.RelPath)
		fmt.Printf("Device: %s | %s\n", cfg.Name, cfg.DeviceType)
		fmt.Printf("Endpoint: %s:%d\n", cfg.ListenAddress, cfg.Port)
		fmt.Printf("Unit IDs: %v\n", cfg.UnitIDs)
		fmt.Printf("Network: protocol=%s data_bits=%d parity=%s stop_bits=%d\n",
			cfg.Network.Protocol,
			cfg.Network.DataBits,
			cfg.Network.Parity,
			cfg.Network.StopBits,
		)
		fmt.Println()
		fmt.Println("1. Start server")
		fmt.Println("2. Change network parameters")
		fmt.Println("3. Save config")
		fmt.Println("4. Choose another config")
		fmt.Println("5. Quit")
		fmt.Print("Choose an option: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return modbus.Config{}, "", false, fmt.Errorf("read config action: %w", err)
		}

		switch strings.TrimSpace(line) {
		case "1":
			return cfg, selected.Path, false, nil
		case "2":
			if err := editNetworkParameters(reader, &cfg); err != nil {
				return modbus.Config{}, "", false, err
			}
		case "3":
			if err := modbus.SaveConfig(selected.Path, cfg); err != nil {
				fmt.Printf("Save failed: %v\n", err)
				continue
			}
			fmt.Printf("Config saved to %s\n", selected.Path)
		case "4":
			return modbus.Config{}, "", true, nil
		case "5", "q", "Q":
			return modbus.Config{}, "", false, fmt.Errorf("startup cancelled")
		default:
			fmt.Println("Invalid selection.")
		}
	}
}

func editNetworkParameters(reader *bufio.Reader, cfg *modbus.Config) error {
	for {
		printMenuHeader()
		fmt.Println("Network Parameters")
		fmt.Printf("1. Listen address [%s]\n", cfg.ListenAddress)
		fmt.Printf("2. Port [%d]\n", cfg.Port)
		fmt.Printf("3. Unit IDs [%s]\n", joinUnitIDs(cfg.UnitIDs))
		fmt.Printf("4. Protocol [%s]\n", cfg.Network.Protocol)
		fmt.Printf("5. Data bits [%d]\n", cfg.Network.DataBits)
		fmt.Printf("6. Parity [%s]\n", cfg.Network.Parity)
		fmt.Printf("7. Stop bits [%d]\n", cfg.Network.StopBits)
		fmt.Println("8. Back")
		fmt.Print("Choose a field to change: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read network menu input: %w", err)
		}

		switch strings.TrimSpace(line) {
		case "1":
			if err := updateListenAddress(reader, cfg); err != nil {
				return err
			}
		case "2":
			if err := updatePort(reader, cfg); err != nil {
				return err
			}
		case "3":
			if err := updateUnitIDs(reader, cfg); err != nil {
				return err
			}
		case "4":
			if err := updateProtocol(reader, cfg); err != nil {
				return err
			}
		case "5":
			if err := updateDataBits(reader, cfg); err != nil {
				return err
			}
		case "6":
			if err := updateParity(reader, cfg); err != nil {
				return err
			}
		case "7":
			if err := updateStopBits(reader, cfg); err != nil {
				return err
			}
		case "8":
			return nil
		default:
			fmt.Println("Invalid selection.")
		}
	}
}

func runRuntimeMenu(ctx context.Context, stop context.CancelFunc, server *modbus.Server, sink *modbus.MemoryLogSink, logger *log.Logger, serverErr <-chan error) error {
	reader := bufio.NewReader(os.Stdin)
	cfg := server.Config()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-serverErr:
			if err != nil {
				return err
			}
			return nil
		default:
		}

		printMenuHeader()
		fmt.Println()
		fmt.Printf("Server: %s | %s | %s:%d\n", cfg.Name, cfg.DeviceType, cfg.ListenAddress, cfg.Port)
		fmt.Println("1. Log")
		fmt.Println("2. Export log")
		fmt.Println("3. Register Map")
		fmt.Println("4. Exit")
		fmt.Print("Choose an option: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read runtime menu input: %w", err)
		}

		switch strings.TrimSpace(line) {
		case "1":
			lines := sink.Snapshot()
			if len(lines) == 0 {
				fmt.Println("Log is empty.")
				continue
			}
			fmt.Println("Log")
			for _, entry := range lines {
				fmt.Println(entry)
			}
		case "2":
			defaultPath := modbus.DefaultLogExportPath()
			fmt.Printf("Export path [%s]: ", defaultPath)
			exportPath, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read export path: %w", err)
			}
			exportPath = strings.TrimSpace(exportPath)
			if exportPath == "" {
				exportPath = defaultPath
			}
			if err := sink.Export(exportPath); err != nil {
				fmt.Printf("Export failed: %v\n", err)
				continue
			}
			fmt.Printf("Log exported to %s\n", exportPath)
		case "3":
			fmt.Println(server.RenderRegisterMap())
		case "4":
			logger.Printf("operator requested shutdown")
			stop()
			return context.Canceled
		default:
			fmt.Println("Invalid selection.")
		}
	}
}

func printMenuHeader() {
	fmt.Println()
	fmt.Println(appTitle)
	fmt.Println(strings.Repeat("=", len(appTitle)))
}

func joinUnitIDs(unitIDs []int) string {
	if len(unitIDs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(unitIDs))
	for _, id := range unitIDs {
		parts = append(parts, strconv.Itoa(id))
	}
	return strings.Join(parts, ", ")
}

func prompt(reader *bufio.Reader, label, current string) (string, error) {
	fmt.Printf("%s [%s]: ", label, current)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input := strings.TrimSpace(line)
	if input == "" {
		return current, nil
	}
	return input, nil
}

func updateListenAddress(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.ListenAddress
	value, err := prompt(reader, "Listen address", current)
	if err != nil {
		return fmt.Errorf("read listen address: %w", err)
	}
	cfg.ListenAddress = value
	return validateEditedConfig(cfg, func() { cfg.ListenAddress = current })
}

func updatePort(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.Port
	value, err := prompt(reader, "Port", strconv.Itoa(current))
	if err != nil {
		return fmt.Errorf("read port: %w", err)
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		fmt.Println("Invalid port.")
		return nil
	}
	cfg.Port = port
	return validateEditedConfig(cfg, func() { cfg.Port = current })
}

func updateUnitIDs(reader *bufio.Reader, cfg *modbus.Config) error {
	current := append([]int(nil), cfg.UnitIDs...)
	value, err := prompt(reader, "Unit IDs (comma-separated)", joinUnitIDs(current))
	if err != nil {
		return fmt.Errorf("read unit ids: %w", err)
	}
	parts := strings.Split(value, ",")
	updated := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, convErr := strconv.Atoi(part)
		if convErr != nil {
			fmt.Println("Invalid unit ID list.")
			return nil
		}
		updated = append(updated, id)
	}
	cfg.UnitIDs = updated
	return validateEditedConfig(cfg, func() { cfg.UnitIDs = current })
}

func updateProtocol(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.Network.Protocol
	value, err := prompt(reader, "Protocol", current)
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	cfg.Network.Protocol = value
	return validateEditedConfig(cfg, func() { cfg.Network.Protocol = current })
}

func updateDataBits(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.Network.DataBits
	value, err := prompt(reader, "Data bits", strconv.Itoa(current))
	if err != nil {
		return fmt.Errorf("read data bits: %w", err)
	}
	bits, err := strconv.Atoi(value)
	if err != nil {
		fmt.Println("Invalid data bits.")
		return nil
	}
	cfg.Network.DataBits = bits
	return validateEditedConfig(cfg, func() { cfg.Network.DataBits = current })
}

func updateParity(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.Network.Parity
	value, err := prompt(reader, "Parity", current)
	if err != nil {
		return fmt.Errorf("read parity: %w", err)
	}
	cfg.Network.Parity = value
	return validateEditedConfig(cfg, func() { cfg.Network.Parity = current })
}

func updateStopBits(reader *bufio.Reader, cfg *modbus.Config) error {
	current := cfg.Network.StopBits
	value, err := prompt(reader, "Stop bits", strconv.Itoa(current))
	if err != nil {
		return fmt.Errorf("read stop bits: %w", err)
	}
	stopBits, err := strconv.Atoi(value)
	if err != nil {
		fmt.Println("Invalid stop bits.")
		return nil
	}
	cfg.Network.StopBits = stopBits
	return validateEditedConfig(cfg, func() { cfg.Network.StopBits = current })
}

func validateEditedConfig(cfg *modbus.Config, revert func()) error {
	if err := cfg.Validate(); err != nil {
		revert()
		fmt.Printf("Change rejected: %v\n", err)
	}
	return nil
}
