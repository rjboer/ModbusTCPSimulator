package modbus

import (
	"fmt"
	"strings"
	"sync"
)

type DataStore struct {
	mu sync.RWMutex

	deviceProfile  string
	runtime        Runtime
	coils          BoolBlock
	discreteInputs BoolBlock
	holding        RegisterBlock
	input          RegisterBlock
}

type DataStoreSnapshot struct {
	Coils          BoolBlock
	DiscreteInputs BoolBlock
	Holding        RegisterBlock
	Input          RegisterBlock
}

func NewDataStore(cfg Config) *DataStore {
	ds := &DataStore{
		deviceProfile:  cfg.DeviceProfile,
		runtime:        cfg.Runtime,
		coils:          cloneBoolBlock(cfg.DataModel.Coils),
		discreteInputs: cloneBoolBlock(cfg.DataModel.DiscreteInputs),
		holding:        cloneRegisterBlock(cfg.DataModel.HoldingRegister),
		input:          cloneRegisterBlock(cfg.DataModel.InputRegisters),
	}
	ds.syncDerivedStateLocked()
	return ds
}

func (ds *DataStore) Snapshot() DataStoreSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return DataStoreSnapshot{
		Coils:          cloneBoolBlock(ds.coils),
		DiscreteInputs: cloneBoolBlock(ds.discreteInputs),
		Holding:        cloneRegisterBlock(ds.holding),
		Input:          cloneRegisterBlock(ds.input),
	}
}

func cloneBoolBlock(src BoolBlock) BoolBlock {
	dst := BoolBlock{
		StartAddress: src.StartAddress,
		Values:       make([]bool, len(src.Values)),
		Descriptions: make([]string, len(src.Descriptions)),
	}
	copy(dst.Values, src.Values)
	copy(dst.Descriptions, src.Descriptions)
	return dst
}

func cloneRegisterBlock(src RegisterBlock) RegisterBlock {
	dst := RegisterBlock{
		StartAddress: src.StartAddress,
		Values:       make([]uint16, len(src.Values)),
		Descriptions: make([]string, len(src.Descriptions)),
	}
	copy(dst.Values, src.Values)
	copy(dst.Descriptions, src.Descriptions)
	return dst
}

func (ds *DataStore) ReadCoils(address, quantity uint16) ([]bool, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return readBoolBlock(ds.coils, address, quantity)
}

func (ds *DataStore) ReadDiscreteInputs(address, quantity uint16) ([]bool, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return readBoolBlock(ds.discreteInputs, address, quantity)
}

func (ds *DataStore) ReadHoldingRegisters(address, quantity uint16) ([]uint16, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return readRegisterBlock(ds.holding, address, quantity)
}

func (ds *DataStore) ReadInputRegisters(address, quantity uint16) ([]uint16, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return readRegisterBlock(ds.input, address, quantity)
}

func (ds *DataStore) WriteSingleCoil(address uint16, value bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return writeSingleBool(&ds.coils, address, value)
}

func (ds *DataStore) WriteSingleRegister(address uint16, value uint16) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if err := writeSingleRegister(&ds.holding, address, value); err != nil {
		return err
	}
	ds.syncDerivedStateLocked()
	return nil
}

func (ds *DataStore) WriteMultipleCoils(address uint16, values []bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return writeMultipleBools(&ds.coils, address, values)
}

func (ds *DataStore) WriteMultipleRegisters(address uint16, values []uint16) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if err := writeMultipleRegisters(&ds.holding, address, values); err != nil {
		return err
	}
	ds.syncDerivedStateLocked()
	return nil
}

func (ds *DataStore) SetDiscreteInput(address uint16, value bool) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return writeSingleBool(&ds.discreteInputs, address, value)
}

func (ds *DataStore) SetInputRegister(address uint16, value uint16) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return writeSingleRegister(&ds.input, address, value)
}

func (ds *DataStore) syncDerivedStateLocked() {
	model := strings.TrimSpace(ds.runtime.Behavior.Model)
	if model == "" && strings.EqualFold(ds.deviceProfile, "delta-ms300") {
		model = "delta-ms300-basic"
	}
	switch strings.ToLower(model) {
	case "delta-ms300-basic":
		ds.syncMS300Locked(ds.runtime.Behavior)
	case "abb-fena-basic":
		ds.syncABBFENALocked(ds.runtime.Behavior)
	case "danfoss-fc302-basic":
		ds.syncDanfossFC302Locked(ds.runtime.Behavior)
	case "invertek-optidrive-p2-basic":
		ds.syncInvertekOptidriveP2Locked(ds.runtime.Behavior)
	case "schneider-atv320-basic":
		ds.syncSchneiderATV320Locked(ds.runtime.Behavior)
	case "siemens-sinamics-basic":
		ds.syncSiemensSINAMICSLocked(ds.runtime.Behavior)
	}
}

func (ds *DataStore) syncMS300Locked(behavior BehaviorConfig) {
	commandAddr := behaviorAddress(behavior, "command", 0x2000)
	frequencyCommandAddr := behaviorAddress(behavior, "frequency_command", 0x2001)

	command, ok := registerValue(ds.holding, commandAddr)
	if !ok {
		return
	}
	freqCommand, ok := registerValue(ds.holding, frequencyCommandAddr)
	if !ok {
		return
	}

	runBits := command & 0x0003
	directionBits := (command >> 4) & 0x0003
	isRunning := runBits == 0x0002 || runBits == 0x0003

	status := behaviorConstant(behavior, "stopped_status_word", 0x0500)
	if isRunning {
		status |= behaviorConstant(behavior, "running_status_bits", 0x0003)
		if directionBits == 0x0002 {
			status |= behaviorConstant(behavior, "reverse_status_bits", 0x0018)
		}
	}

	outputFreq := uint16(0)
	outputVoltage := uint16(0)
	outputCurrent := uint16(0)
	actualSpeed := uint16(0)
	outputTorque := uint16(0)
	powerAngle := uint16(0)
	powerOutput := uint16(0)
	displayCurrent := uint16(0)
	displayOutputFreq := uint16(0)
	displayPowerOutput := uint16(0)
	displayTorqueSigned := uint16(0)
	carrierFrequency := behaviorConstant(behavior, "carrier_frequency", 8)
	magneticPoleZone := behaviorConstant(behavior, "magnetic_pole_zone", 4)
	busRipple := behaviorConstant(behavior, "dc_bus_voltage_ripples", 25)
	driveStatus := behaviorConstant(behavior, "stopped_drive_status", 0x0004)
	masterFrequency := uint16(0)

	if isRunning {
		outputFreq = freqCommand
		outputVoltage = uint16((uint32(freqCommand) * 400) / 5000)
		outputCurrent = uint16(maxUint32(uint32(behaviorConstant(behavior, "minimum_running_current", 100)), (uint32(freqCommand)*25)/1000))
		actualSpeed = uint16((uint32(freqCommand) * 3) / 10)
		outputTorque = behaviorConstant(behavior, "running_torque", 350)
		powerAngle = behaviorConstant(behavior, "running_power_angle", 300)
		powerOutput = uint16((uint32(outputVoltage) * uint32(outputCurrent)) / 1000)
		displayCurrent = outputCurrent
		displayOutputFreq = outputFreq
		displayPowerOutput = powerOutput
		displayTorqueSigned = outputTorque
		masterFrequency = outputFreq
		driveStatus = behaviorConstant(behavior, "running_forward_drive_status", 0x0015)
		if directionBits == 0x0002 {
			driveStatus = behaviorConstant(behavior, "running_reverse_drive_status", 0x0016)
			displayTorqueSigned = 65535 - outputTorque + 1
		}
	}

	setInputRegisterIfMapped(&ds.input, behavior, "fault_status", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "drive_operation_status", status)
	setInputRegisterIfMapped(&ds.input, behavior, "frequency_command_readback", freqCommand)
	setInputRegisterIfMapped(&ds.input, behavior, "output_frequency", outputFreq)
	setInputRegisterIfMapped(&ds.input, behavior, "output_current", outputCurrent)
	setInputRegisterIfMapped(&ds.input, behavior, "dc_bus_voltage", behaviorConstant(behavior, "dc_bus_voltage", 3250))
	setInputRegisterIfMapped(&ds.input, behavior, "output_voltage", outputVoltage)
	setInputRegisterIfMapped(&ds.input, behavior, "multi_step_speed_status", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "counter_value", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "output_power_factor_angle", powerAngle)
	setInputRegisterIfMapped(&ds.input, behavior, "output_torque", outputTorque)
	setInputRegisterIfMapped(&ds.input, behavior, "motor_actual_speed", actualSpeed)
	setInputRegisterIfMapped(&ds.input, behavior, "power_output", powerOutput)
	setInputRegisterIfMapped(&ds.input, behavior, "maximum_user_defined_value", behaviorConstant(behavior, "maximum_user_defined_value", 5000))
	setInputRegisterIfMapped(&ds.input, behavior, "output_current_digit_metadata", behaviorConstant(behavior, "output_current_digit_metadata", 0x0200))
	setInputRegisterIfMapped(&ds.input, behavior, "display_output_current", displayCurrent)
	setInputRegisterIfMapped(&ds.input, behavior, "display_counter_value", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "display_output_frequency", displayOutputFreq)
	setInputRegisterIfMapped(&ds.input, behavior, "display_dc_bus_voltage", behaviorConstant(behavior, "dc_bus_voltage", 3250))
	setInputRegisterIfMapped(&ds.input, behavior, "display_output_voltage", outputVoltage)
	setInputRegisterIfMapped(&ds.input, behavior, "display_power_factor_angle", powerAngle)
	setInputRegisterIfMapped(&ds.input, behavior, "display_power_output", displayPowerOutput)
	setInputRegisterIfMapped(&ds.input, behavior, "display_motor_actual_speed", actualSpeed)
	setInputRegisterIfMapped(&ds.input, behavior, "display_output_torque", outputTorque)
	setInputRegisterIfMapped(&ds.input, behavior, "pid_feedback_value", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "avi_analog_input", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "aci_analog_input", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "igbt_temperature", behaviorConstant(behavior, "igbt_temperature", 350))
	setInputRegisterIfMapped(&ds.input, behavior, "digital_input_status", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "digital_output_status", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "multi_step_speed", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "cpu_pin_status_digital_input", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "cpu_pin_status_digital_output", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "pulse_input_frequency", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "overload_counter", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "gff", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "dc_bus_voltage_ripples", busRipple)
	setInputRegisterIfMapped(&ds.input, behavior, "plc_register_d1043_data", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "magnetic_pole_zone", magneticPoleZone)
	setInputRegisterIfMapped(&ds.input, behavior, "display_user_defined_output", displayOutputFreq)
	setInputRegisterIfMapped(&ds.input, behavior, "pr_00_05_gain_value", displayOutputFreq)
	setInputRegisterIfMapped(&ds.input, behavior, "control_mode", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "carrier_frequency", carrierFrequency)
	setInputRegisterIfMapped(&ds.input, behavior, "drive_status", driveStatus)
	setInputRegisterIfMapped(&ds.input, behavior, "signed_torque", displayTorqueSigned)
	setInputRegisterIfMapped(&ds.input, behavior, "torque_command", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "kwh", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "pid_reference", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "pid_offset", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "pid_output_frequency", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "auxiliary_frequency", 0)
	setInputRegisterIfMapped(&ds.input, behavior, "master_frequency", masterFrequency)
	setInputRegisterIfMapped(&ds.input, behavior, "combined_frequency", masterFrequency)
}

func behaviorAddress(behavior BehaviorConfig, key string, fallback uint16) uint16 {
	if address, ok := behavior.RegisterMap[key]; ok {
		return address
	}
	return fallback
}

func behaviorConstant(behavior BehaviorConfig, key string, fallback uint16) uint16 {
	if value, ok := behavior.Constants[key]; ok {
		return value
	}
	return fallback
}

func (ds *DataStore) syncABBFENALocked(behavior BehaviorConfig) {
	control := registerValueOrZero(ds.holding, behaviorAddress(behavior, "control_word", 1))
	reference1 := registerValueOrZero(ds.holding, behaviorAddress(behavior, "reference_1", 2))
	reference2 := registerValueOrZero(ds.holding, behaviorAddress(behavior, "reference_2", 3))
	dataOut1 := registerValueOrZero(ds.holding, behaviorAddress(behavior, "data_out_1", 4))
	dataOut2 := registerValueOrZero(ds.holding, behaviorAddress(behavior, "data_out_2", 5))

	status := behaviorConstant(behavior, "ready_status_word", 1)
	if control != 0 {
		status = behaviorConstant(behavior, "running_status_word", 3)
	}

	ds.setMappedRegisterIfPresent(behavior, "status_word", status)
	ds.setMappedRegisterIfPresent(behavior, "actual_1", reference1)
	ds.setMappedRegisterIfPresent(behavior, "actual_2", reference2)
	ds.setMappedRegisterIfPresent(behavior, "data_in_1", dataOut1)
	ds.setMappedRegisterIfPresent(behavior, "data_in_2", dataOut2)
}

func (ds *DataStore) syncDanfossFC302Locked(behavior BehaviorConfig) {
	controlWord := registerValueOrZero(ds.holding, behaviorAddress(behavior, "control_word", 2810))
	mainReference := registerValueOrZero(ds.holding, behaviorAddress(behavior, "main_reference", 2811))

	statusWord := behaviorConstant(behavior, "stopped_status_word", 1)
	if controlWord != 0 {
		statusWord = behaviorConstant(behavior, "running_status_word", 3)
	}

	ds.setMappedRegisterIfPresent(behavior, "status_word", statusWord)
	ds.setMappedRegisterIfPresent(behavior, "main_actual_value", mainReference)

	for i := 3; i <= 10; i++ {
		writeKey := fmt.Sprintf("process_data_write_%d", i)
		readKey := fmt.Sprintf("process_data_read_%d", i)
		value := registerValueOrZero(ds.holding, behaviorAddress(behavior, writeKey, 0))
		ds.setMappedRegisterIfPresent(behavior, readKey, value)
	}
}

func (ds *DataStore) syncInvertekOptidriveP2Locked(behavior BehaviorConfig) {
	controlWord := registerValueOrZero(ds.holding, behaviorAddress(behavior, "control_word", 1))
	frequencySetpoint := registerValueOrZero(ds.holding, behaviorAddress(behavior, "frequency_setpoint", 2))
	torqueSetpoint := registerValueOrZero(ds.holding, behaviorAddress(behavior, "torque_setpoint", 3))

	statusWord := behaviorConstant(behavior, "ready_status_word", 9)
	if controlWord != 0 {
		statusWord = behaviorConstant(behavior, "running_status_word", 11)
	}

	outputCurrent := uint16(0)
	if controlWord != 0 {
		outputCurrent = uint16(maxUint32(uint32(behaviorConstant(behavior, "minimum_output_current", 100)), uint32(frequencySetpoint)/5))
	}

	ds.setMappedRegisterIfPresent(behavior, "status_word", statusWord)
	ds.setMappedRegisterIfPresent(behavior, "output_frequency", frequencySetpoint)
	ds.setMappedRegisterIfPresent(behavior, "output_current", outputCurrent)
	ds.setMappedRegisterIfPresent(behavior, "output_torque", torqueSetpoint)
}

func (ds *DataStore) syncSchneiderATV320Locked(behavior BehaviorConfig) {
	commandWord := registerValueOrZero(ds.holding, behaviorAddress(behavior, "cmd", 8501))
	frequencyReference := registerValueOrZero(ds.holding, behaviorAddress(behavior, "lfr", 8502))

	eta := behaviorConstant(behavior, "stopped_eta", 1)
	if commandWord != 0 {
		eta = behaviorConstant(behavior, "running_eta", 7)
	}

	current := uint16(0)
	torque := uint16(0)
	voltage := uint16(0)
	power := uint16(0)
	if commandWord != 0 {
		current = uint16(maxUint32(uint32(behaviorConstant(behavior, "minimum_current", 50)), uint32(frequencyReference)/4))
		torque = behaviorConstant(behavior, "running_torque", 250)
		voltage = uint16((uint32(frequencyReference) * 400) / 500)
		power = uint16((uint32(voltage) * uint32(current)) / 1000)
	}

	ds.setMappedRegisterIfPresent(behavior, "eta", eta)
	ds.setMappedRegisterIfPresent(behavior, "rfr", frequencyReference)
	ds.setMappedRegisterIfPresent(behavior, "lcr", current)
	ds.setMappedRegisterIfPresent(behavior, "otr", torque)
	ds.setMappedRegisterIfPresent(behavior, "eti", eta)
	ds.setMappedRegisterIfPresent(behavior, "uop", voltage)
	ds.setMappedRegisterIfPresent(behavior, "opr", power)
	ds.setMappedRegisterIfPresent(behavior, "lfrd", frequencyReference)
	ds.setMappedRegisterIfPresent(behavior, "rfrd", frequencyReference)
}

func (ds *DataStore) syncSiemensSINAMICSLocked(behavior BehaviorConfig) {
	controlWord := registerValueOrZero(ds.holding, behaviorAddress(behavior, "control_word", 40100))
	mainSetpoint := registerValueOrZero(ds.holding, behaviorAddress(behavior, "main_setpoint", 40101))

	statusWord := behaviorConstant(behavior, "stopped_status_word", 1)
	if controlWord != 0 {
		statusWord = behaviorConstant(behavior, "running_status_word", 3)
	}

	ds.setMappedRegisterIfPresent(behavior, "status_word", statusWord)
	ds.setMappedRegisterIfPresent(behavior, "main_actual_value", mainSetpoint)
	ds.setMappedRegisterIfPresent(behavior, "reference_speed", mainSetpoint)
	ds.setMappedRegisterIfPresent(behavior, "failure_number", 0)
	ds.setMappedRegisterIfPresent(behavior, "alarm_number", 0)
	ds.setMappedRegisterIfPresent(behavior, "actual_alarm_code", 0)

	for i := 3; i <= 10; i++ {
		writeKey := fmt.Sprintf("process_data_write_%d", i)
		readKey := fmt.Sprintf("process_data_read_%d", i)
		value := registerValueOrZero(ds.holding, behaviorAddress(behavior, writeKey, 0))
		ds.setMappedRegisterIfPresent(behavior, readKey, value)
	}
}

func setInputRegisterIfMapped(block *RegisterBlock, behavior BehaviorConfig, key string, value uint16) {
	address, ok := behavior.RegisterMap[key]
	if !ok {
		return
	}
	_ = setRegisterValue(block, address, value)
}

func registerValueOrZero(block RegisterBlock, address uint16) uint16 {
	value, _ := registerValue(block, address)
	return value
}

func (ds *DataStore) setMappedRegisterIfPresent(behavior BehaviorConfig, key string, value uint16) {
	address, ok := behavior.RegisterMap[key]
	if !ok {
		return
	}
	if err := setRegisterValue(&ds.holding, address, value); err == nil {
		return
	}
	_ = setRegisterValue(&ds.input, address, value)
}

func registerValue(block RegisterBlock, address uint16) (uint16, bool) {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, 1)
	if err != nil {
		return 0, false
	}
	return block.Values[start:end][0], true
}

func setRegisterValue(block *RegisterBlock, address uint16, value uint16) error {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, 1)
	if err != nil {
		return err
	}
	block.Values[start:end][0] = value
	return nil
}

func maxUint32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func readBoolBlock(block BoolBlock, address, quantity uint16) ([]bool, error) {
	if quantity == 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, quantity)
	if err != nil {
		return nil, err
	}
	values := make([]bool, end-start)
	copy(values, block.Values[start:end])
	return values, nil
}

func readRegisterBlock(block RegisterBlock, address, quantity uint16) ([]uint16, error) {
	if quantity == 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, quantity)
	if err != nil {
		return nil, err
	}
	values := make([]uint16, end-start)
	copy(values, block.Values[start:end])
	return values, nil
}

func writeSingleBool(block *BoolBlock, address uint16, value bool) error {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, 1)
	if err != nil {
		return err
	}
	block.Values[start:end][0] = value
	return nil
}

func writeSingleRegister(block *RegisterBlock, address uint16, value uint16) error {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, 1)
	if err != nil {
		return err
	}
	block.Values[start:end][0] = value
	return nil
}

func writeMultipleBools(block *BoolBlock, address uint16, values []bool) error {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, uint16(len(values)))
	if err != nil {
		return err
	}
	copy(block.Values[start:end], values)
	return nil
}

func writeMultipleRegisters(block *RegisterBlock, address uint16, values []uint16) error {
	start, end, err := blockRange(block.StartAddress, len(block.Values), address, uint16(len(values)))
	if err != nil {
		return err
	}
	copy(block.Values[start:end], values)
	return nil
}

func blockRange(startAddress uint16, blockLen int, address, quantity uint16) (int, int, error) {
	if address < startAddress {
		return 0, 0, fmt.Errorf("address %d before block start %d", address, startAddress)
	}
	offset := int(address - startAddress)
	end := offset + int(quantity)
	if offset < 0 || end > blockLen {
		return 0, 0, fmt.Errorf("address range %d..%d outside block", address, address+quantity-1)
	}
	return offset, end, nil
}
