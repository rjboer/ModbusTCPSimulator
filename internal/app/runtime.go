package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"modbustcpipserver/internal/modbus"
)

type Runtime struct {
	mu           sync.Mutex
	root         string
	discovery    modbus.DiscoveryResult
	selectedPath string
	selectedCfg  modbus.Config
	logSink      *modbus.MemoryLogSink
	logger       *log.Logger
	server       *modbus.Server
	cancel       context.CancelFunc
	done         chan error
	lastAsyncErr error
}

func NewRuntime(root string) *Runtime {
	logSink := modbus.NewMemoryLogSink()
	logger := log.New(io.MultiWriter(os.Stdout, logSink), "mock-modbus ", log.LstdFlags|log.Lmicroseconds)

	return &Runtime{
		root:    root,
		logSink: logSink,
		logger:  logger,
	}
}

func (r *Runtime) Root() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.root
}

func (r *Runtime) SetRoot(root string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.root = root
	r.discovery = modbus.DiscoveryResult{}
	r.selectedPath = ""
	r.selectedCfg = modbus.Config{}
}

func (r *Runtime) Discover() (modbus.DiscoveryResult, error) {
	discovery, err := modbus.DiscoverConfigs(r.root)
	if err != nil {
		return modbus.DiscoveryResult{}, fmt.Errorf("scan configs in %s: %w", r.root, err)
	}

	if len(discovery.Valid) == 0 {
		defaultPath := filepath.Join(r.root, "config.default_c.json")
		cfg, rendered, created, err := modbus.LoadOrCreateConfig(defaultPath)
		if err != nil {
			return modbus.DiscoveryResult{}, err
		}
		if created {
			r.logger.Printf("no valid JSON server configs found under %s", r.root)
			r.logger.Printf("created default profile config at %q", defaultPath)
			r.logger.Printf("generated config:\n%s", rendered)
		}

		discovery.Valid = append(discovery.Valid, modbus.DiscoveredConfig{
			Path:     defaultPath,
			RelPath:  "config.default_c.json",
			Config:   cfg,
			Rendered: rendered,
		})
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.discovery = discovery
	if r.selectedPath == "" && len(discovery.Valid) > 0 {
		r.selectedPath = discovery.Valid[0].Path
		r.selectedCfg = discovery.Valid[0].Config
	}
	return discovery, nil
}

func (r *Runtime) Discovery() modbus.DiscoveryResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.discovery
}

func (r *Runtime) SelectConfig(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.discovery.Valid {
		if item.Path == path {
			r.selectedPath = item.Path
			r.selectedCfg = item.Config
			return nil
		}
	}
	return fmt.Errorf("config not found: %s", path)
}

func (r *Runtime) SelectedConfig() (string, modbus.Config, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.selectedPath == "" {
		return "", modbus.Config{}, false
	}
	return r.selectedPath, r.selectedCfg, true
}

func (r *Runtime) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.server != nil {
		return fmt.Errorf("server already running")
	}
	if r.selectedPath == "" {
		return fmt.Errorf("no config selected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := modbus.NewServer(r.selectedCfg, r.logger)
	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe(ctx)
	}()

	r.logger.Printf("selected config: %s", r.selectedPath)
	r.server = server
	r.cancel = cancel
	r.done = done
	r.lastAsyncErr = nil
	return nil
}

func (r *Runtime) Stop() error {
	r.mu.Lock()
	r.syncServerLocked()
	if r.server == nil {
		r.mu.Unlock()
		return nil
	}

	cancel := r.cancel
	done := r.done
	r.logger.Printf("operator requested shutdown")
	r.server = nil
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()

	cancel()
	if done != nil {
		if err := <-done; err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.syncServerLocked()
	return r.server != nil
}

func (r *Runtime) SyncState() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.syncServerLocked()
	err := r.lastAsyncErr
	r.lastAsyncErr = nil
	return r.server != nil, err
}

func (r *Runtime) LogsSince(offset int) ([]string, int) {
	return r.logSink.SnapshotSince(offset)
}

func (r *Runtime) ExportLogs(path string) error {
	return r.logSink.Export(path)
}

func (r *Runtime) RegisterMap() string {
	r.mu.Lock()
	r.syncServerLocked()
	server := r.server
	cfg := r.selectedCfg
	r.mu.Unlock()

	if server != nil {
		return server.RenderRegisterMap()
	}
	return modbus.RenderRegisterMap(cfg, modbus.NewDataStore(cfg).Snapshot())
}

func (r *Runtime) TrafficSnapshot() modbus.TrafficSnapshot {
	r.mu.Lock()
	r.syncServerLocked()
	server := r.server
	r.mu.Unlock()

	if server == nil {
		return modbus.TrafficSnapshot{}
	}
	return server.TrafficSnapshot()
}

func (r *Runtime) SnapshotData() (modbus.Config, modbus.DataStoreSnapshot) {
	r.mu.Lock()
	r.syncServerLocked()
	server := r.server
	cfg := r.selectedCfg
	r.mu.Unlock()

	if server != nil {
		return server.Config(), server.Snapshot()
	}

	store := modbus.NewDataStore(cfg)
	return cfg, store.Snapshot()
}

func (r *Runtime) SetBoolValue(block string, address uint16, value bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.syncServerLocked()

	switch block {
	case "Coils":
		if err := setConfigBoolBlockValue(&r.selectedCfg.DataModel.Coils, address, value); err != nil {
			return err
		}
		if r.server != nil {
			if err := r.server.SetCoil(address, value); err != nil {
				return err
			}
		}
	case "Discrete Inputs":
		if err := setConfigBoolBlockValue(&r.selectedCfg.DataModel.DiscreteInputs, address, value); err != nil {
			return err
		}
		if r.server != nil {
			if err := r.server.SetDiscreteInput(address, value); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("block %q does not support boolean editing", block)
	}

	r.logger.Printf("operator set %s address=%d value=%t", block, address, value)
	return nil
}

func (r *Runtime) SetRegisterValue(block string, address uint16, value uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.syncServerLocked()

	switch block {
	case "Holding Registers":
		if err := setConfigRegisterBlockValue(&r.selectedCfg.DataModel.HoldingRegister, address, value); err != nil {
			return err
		}
		if r.server != nil {
			if err := r.server.SetHoldingRegister(address, value); err != nil {
				return err
			}
		}
	case "Input Registers":
		if err := setConfigRegisterBlockValue(&r.selectedCfg.DataModel.InputRegisters, address, value); err != nil {
			return err
		}
		if r.server != nil {
			if err := r.server.SetInputRegister(address, value); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("block %q does not support register editing", block)
	}

	r.logger.Printf("operator set %s address=%d value=%d (0x%04X)", block, address, value, value)
	return nil
}

func (r *Runtime) syncServerLocked() {
	if r.done == nil {
		return
	}

	select {
	case err := <-r.done:
		if err != nil {
			r.logger.Printf("server stopped with error: %v", err)
			r.lastAsyncErr = err
		}
		r.server = nil
		r.cancel = nil
		r.done = nil
	default:
	}
}

func setConfigBoolBlockValue(block *modbus.BoolBlock, address uint16, value bool) error {
	if address < block.StartAddress {
		return fmt.Errorf("address %d before block start %d", address, block.StartAddress)
	}
	index := int(address - block.StartAddress)
	if index < 0 || index >= len(block.Values) {
		return fmt.Errorf("address %d outside block", address)
	}
	block.Values[index] = value
	return nil
}

func setConfigRegisterBlockValue(block *modbus.RegisterBlock, address uint16, value uint16) error {
	if address < block.StartAddress {
		return fmt.Errorf("address %d before block start %d", address, block.StartAddress)
	}
	index := int(address - block.StartAddress)
	if index < 0 || index >= len(block.Values) {
		return fmt.Errorf("address %d outside block", address)
	}
	block.Values[index] = value
	return nil
}
