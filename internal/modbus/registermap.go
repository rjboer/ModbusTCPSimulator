package modbus

import (
	"fmt"
	"strings"
)

func RenderRegisterMap(cfg Config, snapshot DataStoreSnapshot) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Device: %s\n", cfg.Name)
	fmt.Fprintf(&b, "Device type: %s\n", cfg.DeviceType)
	fmt.Fprintf(&b, "Profile: %s\n", cfg.DeviceProfile)
	fmt.Fprintf(&b, "Endpoint: %s:%d\n", cfg.ListenAddress, cfg.Port)
	fmt.Fprintf(&b, "Unit IDs: %v\n", cfg.UnitIDs)
	fmt.Fprintf(&b, "Network: protocol=%s data_bits=%d parity=%s stop_bits=%d\n",
		cfg.Network.Protocol,
		cfg.Network.DataBits,
		cfg.Network.Parity,
		cfg.Network.StopBits,
	)
	b.WriteString("\nRegister blocks\n")
	renderBoolBlock(&b, "Coils", snapshot.Coils)
	renderBoolBlock(&b, "Discrete inputs", snapshot.DiscreteInputs)
	renderRegisterBlock(&b, "Holding registers", snapshot.Holding)
	renderRegisterBlock(&b, "Input registers", snapshot.Input)
	return b.String()
}

func renderBoolBlock(b *strings.Builder, label string, block BoolBlock) {
	fmt.Fprintf(b, "\n%s\n", label)
	fmt.Fprintf(b, "  Start: %d (0x%04X)\n", block.StartAddress, block.StartAddress)
	fmt.Fprintf(b, "  Count: %d\n", len(block.Values))
	for i, value := range block.Values {
		address := block.StartAddress + uint16(i)
		description := descriptionAt(block.Descriptions, i)
		if description != "" {
			fmt.Fprintf(b, "  %5d (0x%04X): %t | %s\n", address, address, value, description)
			continue
		}
		fmt.Fprintf(b, "  %5d (0x%04X): %t\n", address, address, value)
	}
}

func renderRegisterBlock(b *strings.Builder, label string, block RegisterBlock) {
	fmt.Fprintf(b, "\n%s\n", label)
	fmt.Fprintf(b, "  Start: %d (0x%04X)\n", block.StartAddress, block.StartAddress)
	fmt.Fprintf(b, "  Count: %d\n", len(block.Values))
	for i, value := range block.Values {
		address := block.StartAddress + uint16(i)
		description := descriptionAt(block.Descriptions, i)
		if description != "" {
			fmt.Fprintf(b, "  %5d (0x%04X): %5d (0x%04X) | %s\n", address, address, value, value, description)
			continue
		}
		fmt.Fprintf(b, "  %5d (0x%04X): %5d (0x%04X)\n", address, address, value, value)
	}
}

func descriptionAt(descriptions []string, index int) string {
	if index < 0 || index >= len(descriptions) {
		return ""
	}
	return descriptions[index]
}
