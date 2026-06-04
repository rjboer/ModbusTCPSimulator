package modbus

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Config struct {
	Name          string        `json:"name"`
	DeviceType    string        `json:"device_type"`
	DeviceProfile string        `json:"device_profile"`
	ListenAddress string        `json:"listen_address"`
	Port          int           `json:"port"`
	UnitIDs       []int         `json:"unit_ids"`
	LogLevel      string        `json:"log_level"`
	Network       Network       `json:"network"`
	Connection    Connection    `json:"connection"`
	Documentation Documentation `json:"documentation,omitempty"`
	DataModel     DataModel     `json:"data_model"`
	Runtime       Runtime       `json:"runtime,omitempty"`
}

type Network struct {
	Protocol string `json:"protocol"`
	DataBits int    `json:"data_bits"`
	Parity   string `json:"parity"`
	StopBits int    `json:"stop_bits"`
}

type Connection struct {
	MaxClients    int `json:"max_clients"`
	IdleTimeoutMs int `json:"idle_timeout_ms"`
}

type Documentation struct {
	Trust              string            `json:"trust,omitempty"`
	LocalSources       []string          `json:"local_sources,omitempty"`
	OfficialReferences []string          `json:"official_references,omitempty"`
	NetworkValidation  map[string]string `json:"network_validation,omitempty"`
	Notes              []string          `json:"notes,omitempty"`
}

type DataModel struct {
	Coils           BoolBlock     `json:"coils"`
	DiscreteInputs  BoolBlock     `json:"discrete_inputs"`
	HoldingRegister RegisterBlock `json:"holding_registers"`
	InputRegisters  RegisterBlock `json:"input_registers"`
}

type BoolBlock struct {
	StartAddress uint16   `json:"start_address"`
	Values       []bool   `json:"values"`
	Descriptions []string `json:"descriptions,omitempty"`
}

type RegisterBlock struct {
	StartAddress uint16   `json:"start_address"`
	Values       []uint16 `json:"values"`
	Descriptions []string `json:"descriptions,omitempty"`
}

type Runtime struct {
	Behavior BehaviorConfig `json:"behavior,omitempty"`
}

type BehaviorConfig struct {
	Model       string            `json:"model,omitempty"`
	RegisterMap map[string]uint16 `json:"register_map,omitempty"`
	Constants   map[string]uint16 `json:"constants,omitempty"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadOrCreateConfig(path string) (Config, string, bool, error) {
	cfg, err := LoadConfig(path)
	if err == nil {
		rendered, marshalErr := MarshalConfig(cfg)
		if marshalErr != nil {
			return Config{}, "", false, marshalErr
		}
		return cfg, rendered, false, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, "", false, err
	}

	cfg = DefaultMS300Config()
	rendered, err := MarshalConfig(cfg)
	if err != nil {
		return Config{}, "", false, fmt.Errorf("render default config: %w", err)
	}
	if writeErr := os.WriteFile(path, []byte(rendered), 0o644); writeErr != nil {
		return Config{}, "", false, fmt.Errorf("write default config: %w", writeErr)
	}
	return cfg, rendered, true, nil
}

func MarshalConfig(cfg Config) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	return string(data) + "\n", nil
}

func SaveConfig(path string, cfg Config) error {
	rendered, err := MarshalConfig(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Name == "" {
		c.Name = "Delta MS300 Mock Drive"
	}
	if c.DeviceType == "" {
		c.DeviceType = "frequency-drive"
	}
	if c.DeviceProfile == "" {
		c.DeviceProfile = "delta-ms300"
	}
	if c.ListenAddress == "" {
		c.ListenAddress = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 502
	}
	if c.Connection.MaxClients == 0 {
		c.Connection.MaxClients = 8
	}
	if c.Connection.IdleTimeoutMs == 0 {
		c.Connection.IdleTimeoutMs = 30000
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Network.Protocol == "" {
		c.Network.Protocol = "modbus-tcp"
	}
	if c.Network.DataBits == 0 {
		c.Network.DataBits = 8
	}
	if c.Network.Parity == "" {
		c.Network.Parity = "none"
	}
	if c.Network.StopBits == 0 {
		c.Network.StopBits = 1
	}
	if c.Runtime.Behavior.Model == "" {
		switch c.DeviceProfile {
		case "delta-ms300":
			c.Runtime.Behavior = DefaultDeltaMS300Behavior()
		case "abb-fena":
			c.Runtime.Behavior = DefaultABBFENABehavior()
		case "danfoss-fc302":
			c.Runtime.Behavior = DefaultDanfossFC302Behavior()
		case "invertek-optidrive-p2":
			c.Runtime.Behavior = DefaultInvertekOptidriveP2Behavior()
		case "schneider-atv320":
			c.Runtime.Behavior = DefaultSchneiderATV320Behavior()
		case "siemens-sinamics":
			c.Runtime.Behavior = DefaultSiemensSINAMICSBehavior()
		}
	}
}

func (c Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if c.DeviceType == "" {
		return fmt.Errorf("device_type must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port %d", c.Port)
	}
	if c.Connection.MaxClients < 1 {
		return fmt.Errorf("max_clients must be >= 1")
	}
	if c.Connection.IdleTimeoutMs < 1 {
		return fmt.Errorf("idle_timeout_ms must be >= 1")
	}
	if c.Network.DataBits < 1 {
		return fmt.Errorf("network.data_bits must be >= 1")
	}
	if c.Network.StopBits < 1 {
		return fmt.Errorf("network.stop_bits must be >= 1")
	}
	if len(c.DataModel.Coils.Descriptions) > 0 && len(c.DataModel.Coils.Descriptions) != len(c.DataModel.Coils.Values) {
		return fmt.Errorf("coils.descriptions length must match coils.values length")
	}
	if len(c.DataModel.DiscreteInputs.Descriptions) > 0 && len(c.DataModel.DiscreteInputs.Descriptions) != len(c.DataModel.DiscreteInputs.Values) {
		return fmt.Errorf("discrete_inputs.descriptions length must match discrete_inputs.values length")
	}
	if len(c.DataModel.HoldingRegister.Descriptions) > 0 && len(c.DataModel.HoldingRegister.Descriptions) != len(c.DataModel.HoldingRegister.Values) {
		return fmt.Errorf("holding_registers.descriptions length must match holding_registers.values length")
	}
	if len(c.DataModel.InputRegisters.Descriptions) > 0 && len(c.DataModel.InputRegisters.Descriptions) != len(c.DataModel.InputRegisters.Values) {
		return fmt.Errorf("input_registers.descriptions length must match input_registers.values length")
	}
	return nil
}

func DefaultMS300Config() Config {
	holding := make([]uint16, 3)
	holding[0x0000] = 0x0000
	holding[0x0001] = 5000
	holding[0x0002] = 0x0000

	input := make([]uint16, 0x135)
	input[0x0000] = 0x0000
	input[0x0001] = 0x0500
	input[0x0002] = 5000
	input[0x0003] = 0
	input[0x0004] = 0
	input[0x0005] = 3250
	input[0x0006] = 0
	input[0x0007] = 0
	input[0x0009] = 0
	input[0x000A] = 0
	input[0x000B] = 0
	input[0x000C] = 0
	input[0x001B] = 5000
	input[0x001F] = 0
	input[0x0100] = 0
	input[0x0102] = 0
	input[0x0103] = 3250
	input[0x0104] = 0
	input[0x0105] = 0
	input[0x0106] = 0
	input[0x0107] = 0
	input[0x0108] = 0
	input[0x010A] = 0
	input[0x010B] = 0
	input[0x010C] = 0
	input[0x010E] = 350
	input[0x0110] = 0
	input[0x0111] = 0
	input[0x0112] = 0
	input[0x0113] = 0
	input[0x0114] = 0
	input[0x0116] = 0
	input[0x0119] = 0
	input[0x011A] = 0
	input[0x011B] = 25
	input[0x011C] = 0
	input[0x011D] = 4
	input[0x011E] = 0
	input[0x011F] = 0
	input[0x0123] = 0
	input[0x0124] = 8
	input[0x0126] = 0x0004
	input[0x0127] = 0
	input[0x0128] = 0
	input[0x0129] = 0
	input[0x012E] = 0
	input[0x012F] = 0
	input[0x0130] = 0
	input[0x0132] = 0
	input[0x0133] = 0
	input[0x0134] = 0

	cfg := Config{
		Name:          "Delta MS300 Mock Drive",
		DeviceType:    "frequency-drive",
		DeviceProfile: "delta-ms300",
		ListenAddress: "127.0.0.1",
		Port:          502,
		UnitIDs:       []int{1},
		LogLevel:      "info",
		Network: Network{
			Protocol: "modbus-tcp",
			DataBits: 8,
			Parity:   "none",
			StopBits: 1,
		},
		Connection: Connection{
			MaxClients:    8,
			IdleTimeoutMs: 30000,
		},
		Runtime: Runtime{
			Behavior: DefaultDeltaMS300Behavior(),
		},
		DataModel: DataModel{
			Coils: BoolBlock{
				StartAddress: 0,
				Values:       []bool{false, false, false, false, false, false, false, false},
				Descriptions: []string{
					"Reserved coil 0",
					"Reserved coil 1",
					"Reserved coil 2",
					"Reserved coil 3",
					"Reserved coil 4",
					"Reserved coil 5",
					"Reserved coil 6",
					"Reserved coil 7",
				},
			},
			DiscreteInputs: BoolBlock{
				StartAddress: 0,
				Values:       []bool{false, false, false, false, false, false, false, false},
				Descriptions: []string{
					"Reserved discrete input 0",
					"Reserved discrete input 1",
					"Reserved discrete input 2",
					"Reserved discrete input 3",
					"Reserved discrete input 4",
					"Reserved discrete input 5",
					"Reserved discrete input 6",
					"Reserved discrete input 7",
				},
			},
			HoldingRegister: RegisterBlock{
				StartAddress: 0x2000,
				Values:       holding,
				Descriptions: []string{
					"Operation command",
					"Frequency command in 0.01 Hz units",
					"Fault and control command source",
				},
			},
			InputRegisters: RegisterBlock{
				StartAddress: 0x2100,
				Values:       input,
				Descriptions: buildMS300InputDescriptions(len(input)),
			},
		},
	}
	return cfg
}

func DefaultDeltaMS300Behavior() BehaviorConfig {
	return BehaviorConfig{
		Model: "delta-ms300-basic",
		RegisterMap: map[string]uint16{
			"command":                       0x2000,
			"frequency_command":             0x2001,
			"fault_status":                  0x2100,
			"drive_operation_status":        0x2101,
			"frequency_command_readback":    0x2102,
			"output_frequency":              0x2103,
			"output_current":                0x2104,
			"dc_bus_voltage":                0x2105,
			"output_voltage":                0x2106,
			"multi_step_speed_status":       0x2107,
			"counter_value":                 0x2109,
			"output_power_factor_angle":     0x210A,
			"output_torque":                 0x210B,
			"motor_actual_speed":            0x210C,
			"power_output":                  0x210F,
			"maximum_user_defined_value":    0x211B,
			"output_current_digit_metadata": 0x211F,
			"display_output_current":        0x2200,
			"display_counter_value":         0x2201,
			"display_output_frequency":      0x2202,
			"display_dc_bus_voltage":        0x2203,
			"display_output_voltage":        0x2204,
			"display_power_factor_angle":    0x2205,
			"display_power_output":          0x2206,
			"display_motor_actual_speed":    0x2207,
			"display_output_torque":         0x2208,
			"pid_feedback_value":            0x220A,
			"avi_analog_input":              0x220B,
			"aci_analog_input":              0x220C,
			"igbt_temperature":              0x220E,
			"digital_input_status":          0x2210,
			"digital_output_status":         0x2211,
			"multi_step_speed":              0x2212,
			"cpu_pin_status_digital_input":  0x2213,
			"cpu_pin_status_digital_output": 0x2214,
			"pulse_input_frequency":         0x2216,
			"overload_counter":              0x2219,
			"gff":                           0x221A,
			"dc_bus_voltage_ripples":        0x221B,
			"plc_register_d1043_data":       0x221C,
			"magnetic_pole_zone":            0x221D,
			"display_user_defined_output":   0x221E,
			"pr_00_05_gain_value":           0x221F,
			"control_mode":                  0x2223,
			"carrier_frequency":             0x2224,
			"drive_status":                  0x2226,
			"signed_torque":                 0x2227,
			"torque_command":                0x2228,
			"kwh":                           0x2229,
			"pid_reference":                 0x222E,
			"pid_offset":                    0x222F,
			"pid_output_frequency":          0x2230,
			"auxiliary_frequency":           0x2232,
			"master_frequency":              0x2233,
			"combined_frequency":            0x2234,
		},
		Constants: map[string]uint16{
			"stopped_status_word":           0x0500,
			"running_status_bits":           0x0003,
			"reverse_status_bits":           0x0018,
			"dc_bus_voltage":                3250,
			"igbt_temperature":              350,
			"dc_bus_voltage_ripples":        25,
			"magnetic_pole_zone":            4,
			"carrier_frequency":             8,
			"maximum_user_defined_value":    5000,
			"output_current_digit_metadata": 0x0200,
			"stopped_drive_status":          0x0004,
			"running_forward_drive_status":  0x0015,
			"running_reverse_drive_status":  0x0016,
			"running_torque":                350,
			"running_power_angle":           300,
			"minimum_running_current":       100,
		},
	}
}

func DefaultABBFENABehavior() BehaviorConfig {
	return BehaviorConfig{
		Model: "abb-fena-basic",
		RegisterMap: map[string]uint16{
			"control_word": 1,
			"reference_1":  2,
			"reference_2":  3,
			"data_out_1":   4,
			"data_out_2":   5,
			"status_word":  51,
			"actual_1":     52,
			"actual_2":     53,
			"data_in_1":    54,
			"data_in_2":    55,
		},
		Constants: map[string]uint16{
			"ready_status_word":   1,
			"running_status_word": 3,
		},
	}
}

func DefaultDanfossFC302Behavior() BehaviorConfig {
	registerMap := map[string]uint16{
		"control_word":      2810,
		"main_reference":    2811,
		"status_word":       2910,
		"main_actual_value": 2911,
	}
	for i := 3; i <= 10; i++ {
		registerMap[fmt.Sprintf("process_data_write_%d", i)] = uint16(2807 + i)
		registerMap[fmt.Sprintf("process_data_read_%d", i)] = uint16(2907 + i)
	}
	return BehaviorConfig{
		Model:       "danfoss-fc302-basic",
		RegisterMap: registerMap,
		Constants: map[string]uint16{
			"stopped_status_word": 1,
			"running_status_word": 3,
		},
	}
}

func DefaultInvertekOptidriveP2Behavior() BehaviorConfig {
	return BehaviorConfig{
		Model: "invertek-optidrive-p2-basic",
		RegisterMap: map[string]uint16{
			"control_word":       1,
			"frequency_setpoint": 2,
			"torque_setpoint":    3,
			"status_word":        256,
			"output_frequency":   257,
			"output_current":     258,
			"output_torque":      259,
		},
		Constants: map[string]uint16{
			"ready_status_word":      9,
			"running_status_word":    11,
			"minimum_output_current": 100,
		},
	}
}

func DefaultSchneiderATV320Behavior() BehaviorConfig {
	return BehaviorConfig{
		Model: "schneider-atv320-basic",
		RegisterMap: map[string]uint16{
			"cmd":  8501,
			"lfr":  8502,
			"pisp": 8503,
			"cmi":  8504,
			"lfrd": 8602,
			"rfrd": 8604,
			"eta":  3201,
			"rfr":  3202,
			"lcr":  3204,
			"otr":  3205,
			"eti":  3206,
			"uop":  3208,
			"opr":  3211,
		},
		Constants: map[string]uint16{
			"stopped_eta":     1,
			"running_eta":     7,
			"minimum_current": 50,
			"running_torque":  250,
		},
	}
}

func DefaultSiemensSINAMICSBehavior() BehaviorConfig {
	registerMap := map[string]uint16{
		"control_word":      40100,
		"main_setpoint":     40101,
		"status_word":       40110,
		"main_actual_value": 40111,
		"reference_speed":   40324,
		"failure_number":    40400,
		"alarm_number":      40408,
		"actual_alarm_code": 40409,
	}
	for i := 3; i <= 10; i++ {
		registerMap[fmt.Sprintf("process_data_write_%d", i)] = uint16(40099 + i)
		registerMap[fmt.Sprintf("process_data_read_%d", i)] = uint16(40109 + i)
	}
	return BehaviorConfig{
		Model:       "siemens-sinamics-basic",
		RegisterMap: registerMap,
		Constants: map[string]uint16{
			"stopped_status_word": 1,
			"running_status_word": 3,
		},
	}
}

func buildMS300InputDescriptions(length int) []string {
	descriptions := make([]string, length)
	for i := range descriptions {
		descriptions[i] = fmt.Sprintf("Reserved input register offset %d", i)
	}
	descriptions[0x0000] = "Fault status"
	descriptions[0x0001] = "Drive operation status"
	descriptions[0x0002] = "Frequency command readback in 0.01 Hz units"
	descriptions[0x0003] = "Output frequency in 0.01 Hz units"
	descriptions[0x0004] = "Output current"
	descriptions[0x0005] = "DC bus voltage"
	descriptions[0x0006] = "Output voltage"
	descriptions[0x0007] = "Multi-step speed status"
	descriptions[0x0009] = "Counter value"
	descriptions[0x000A] = "Output power factor angle"
	descriptions[0x000B] = "Output torque"
	descriptions[0x000C] = "Motor actual speed"
	descriptions[0x001B] = "Maximum user-defined value"
	descriptions[0x001F] = "Output current digit metadata"
	descriptions[0x0100] = "Display output current"
	descriptions[0x0102] = "Display output frequency"
	descriptions[0x0103] = "Display DC bus voltage"
	descriptions[0x0104] = "Display output voltage"
	descriptions[0x0105] = "Display power factor angle"
	descriptions[0x0106] = "Display power output"
	descriptions[0x0107] = "Display motor actual speed"
	descriptions[0x0108] = "Display output torque"
	descriptions[0x010A] = "PID feedback value"
	descriptions[0x010B] = "AVI analog input"
	descriptions[0x010C] = "ACI analog input"
	descriptions[0x010E] = "IGBT temperature"
	descriptions[0x0110] = "Digital input status"
	descriptions[0x0111] = "Digital output status"
	descriptions[0x0112] = "Multi-step speed"
	descriptions[0x0113] = "CPU pin status of digital input"
	descriptions[0x0114] = "CPU pin status of digital output"
	descriptions[0x0116] = "Pulse input frequency"
	descriptions[0x0119] = "Overload counter"
	descriptions[0x011A] = "GFF"
	descriptions[0x011B] = "DC bus voltage ripples"
	descriptions[0x011C] = "PLC register D1043 data"
	descriptions[0x011D] = "Magnetic pole zone"
	descriptions[0x011E] = "Display of user-defined output"
	descriptions[0x011F] = "Pr.00-05 gain value"
	descriptions[0x0123] = "Control mode"
	descriptions[0x0124] = "Frequency of carrier wave"
	descriptions[0x0126] = "Drive status"
	descriptions[0x0127] = "Positive / negative torque"
	descriptions[0x0128] = "Torque command"
	descriptions[0x0129] = "kWh"
	descriptions[0x012E] = "PID reference"
	descriptions[0x012F] = "PID offset"
	descriptions[0x0130] = "PID output frequency"
	descriptions[0x0132] = "Auxiliary frequency"
	descriptions[0x0133] = "Master frequency"
	descriptions[0x0134] = "Frequency after master/auxiliary combination"
	return descriptions
}
