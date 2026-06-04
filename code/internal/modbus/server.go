package modbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

const (
	functionReadCoils            = 0x01
	functionReadDiscreteInputs   = 0x02
	functionReadHoldingRegisters = 0x03
	functionReadInputRegisters   = 0x04
	functionWriteSingleCoil      = 0x05
	functionWriteSingleRegister  = 0x06
	functionWriteMultipleCoils   = 0x0F
	functionWriteMultipleRegs    = 0x10
)

const (
	exceptionIllegalFunction    = 0x01
	exceptionIllegalDataAddress = 0x02
	exceptionIllegalDataValue   = 0x03
	exceptionServerFailure      = 0x04
)

type Server struct {
	cfg          Config
	store        *DataStore
	logger       *log.Logger
	listener     net.Listener
	activeClient atomic.Int32
	traffic      *trafficCollector
}

func NewServer(cfg Config, logger *log.Logger) *Server {
	return &Server{
		cfg:     cfg,
		store:   NewDataStore(cfg),
		logger:  logger,
		traffic: newTrafficCollector(time.Now()),
	}
}

func (s *Server) Config() Config {
	return s.cfg
}

func (s *Server) RenderRegisterMap() string {
	return RenderRegisterMap(s.cfg, s.store.Snapshot())
}

func (s *Server) Snapshot() DataStoreSnapshot {
	return s.store.Snapshot()
}

func (s *Server) ActiveClients() int {
	return int(s.activeClient.Load())
}

func (s *Server) TrafficSnapshot() TrafficSnapshot {
	return s.traffic.snapshot(s.ActiveClients(), time.Now())
}

func (s *Server) SetCoil(address uint16, value bool) error {
	return s.store.WriteSingleCoil(address, value)
}

func (s *Server) SetDiscreteInput(address uint16, value bool) error {
	return s.store.SetDiscreteInput(address, value)
}

func (s *Server) SetHoldingRegister(address uint16, value uint16) error {
	return s.store.WriteSingleRegister(address, value)
}

func (s *Server) SetInputRegister(address uint16, value uint16) error {
	return s.store.SetInputRegister(address, value)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", s.cfg.ListenAddress, s.cfg.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	s.listener = listener
	s.logger.Printf("modbus server listening on %s", address)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				s.logger.Printf("temporary accept error: %v", err)
				continue
			}
			return fmt.Errorf("accept connection: %w", err)
		}

		if int(s.activeClient.Load()) >= s.cfg.Connection.MaxClients {
			s.logger.Printf("rejecting client %s: connection limit reached", conn.RemoteAddr())
			_ = conn.Close()
			continue
		}

		s.activeClient.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	defer func() {
		s.activeClient.Add(-1)
		s.traffic.connectionClosed(remoteAddr)
		_ = conn.Close()
	}()

	s.traffic.connectionOpened(remoteAddr, time.Now())
	s.logger.Printf("client connected: %s", remoteAddr)
	defer s.logger.Printf("client disconnected: %s", remoteAddr)

	idleTimeout := time.Duration(s.cfg.Connection.IdleTimeoutMs) * time.Millisecond
	header := make([]byte, 7)

	for {
		if err := conn.SetDeadline(time.Now().Add(idleTimeout)); err != nil {
			s.logger.Printf("set deadline failed for %s: %v", remoteAddr, err)
			return
		}

		if _, err := io.ReadFull(conn, header); err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, io.EOF) {
				return
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				s.logger.Printf("client idle timeout: %s", remoteAddr)
				return
			}
			s.logger.Printf("read mbap header failed from %s: %v", remoteAddr, err)
			return
		}

		request, err := parseRequest(header, conn)
		if err != nil {
			s.logger.Printf("invalid request from %s: %v", remoteAddr, err)
			return
		}
		s.traffic.recordRead(remoteAddr, 7+len(request.pdu), request.unitID, request.functionCode, time.Now())

		if !s.acceptUnitID(request.unitID) {
			s.logger.Printf("rejected unit id %d from %s", request.unitID, remoteAddr)
			if written, err := writeResponse(conn, request.transactionID, request.unitID, exceptionPDU(request.functionCode, exceptionIllegalDataAddress)); err != nil {
				s.logger.Printf("write exception failed: %v", err)
			} else {
				s.traffic.recordWrite(remoteAddr, written, time.Now())
			}
			continue
		}

		responsePDU := s.handlePDU(request, remoteAddr)
		if written, err := writeResponse(conn, request.transactionID, request.unitID, responsePDU); err != nil {
			s.logger.Printf("write response failed to %s: %v", remoteAddr, err)
			return
		} else {
			s.traffic.recordWrite(remoteAddr, written, time.Now())
		}
	}
}

type requestFrame struct {
	transactionID uint16
	unitID        uint8
	functionCode  uint8
	pdu           []byte
}

func parseRequest(header []byte, conn net.Conn) (requestFrame, error) {
	transactionID := binary.BigEndian.Uint16(header[0:2])
	protocolID := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	unitID := header[6]

	if protocolID != 0 {
		return requestFrame{}, fmt.Errorf("unsupported protocol id %d", protocolID)
	}
	if length < 2 {
		return requestFrame{}, fmt.Errorf("invalid modbus length %d", length)
	}

	pduLen := int(length) - 1
	pdu := make([]byte, pduLen)
	if _, err := io.ReadFull(conn, pdu); err != nil {
		return requestFrame{}, fmt.Errorf("read pdu: %w", err)
	}
	if len(pdu) == 0 {
		return requestFrame{}, fmt.Errorf("empty pdu")
	}

	return requestFrame{
		transactionID: transactionID,
		unitID:        unitID,
		functionCode:  pdu[0],
		pdu:           pdu,
	}, nil
}

func (s *Server) acceptUnitID(unitID uint8) bool {
	if len(s.cfg.UnitIDs) == 0 {
		return true
	}
	for _, allowed := range s.cfg.UnitIDs {
		if allowed >= 0 && allowed <= 255 && uint8(allowed) == unitID {
			return true
		}
	}
	return false
}

func (s *Server) handlePDU(request requestFrame, remoteAddr string) []byte {
	s.logger.Printf("request remote=%s tx=%d unit=%d func=0x%02X payload=% X", remoteAddr, request.transactionID, request.unitID, request.functionCode, request.pdu[1:])

	switch request.functionCode {
	case functionReadCoils:
		return s.handleReadBits(request.pdu, s.store.ReadCoils, 2000)
	case functionReadDiscreteInputs:
		return s.handleReadBits(request.pdu, s.store.ReadDiscreteInputs, 2000)
	case functionReadHoldingRegisters:
		return s.handleReadRegisters(request.pdu, s.store.ReadHoldingRegisters, 125)
	case functionReadInputRegisters:
		return s.handleReadRegisters(request.pdu, s.store.ReadInputRegisters, 125)
	case functionWriteSingleCoil:
		return s.handleWriteSingleCoil(request.pdu)
	case functionWriteSingleRegister:
		return s.handleWriteSingleRegister(request.pdu)
	case functionWriteMultipleCoils:
		return s.handleWriteMultipleCoils(request.pdu)
	case functionWriteMultipleRegs:
		return s.handleWriteMultipleRegisters(request.pdu)
	default:
		return exceptionPDU(request.functionCode, exceptionIllegalFunction)
	}
}

func (s *Server) handleReadBits(pdu []byte, reader func(uint16, uint16) ([]bool, error), maxQuantity uint16) []byte {
	address, quantity, ok := parseReadRequest(pdu)
	if !ok || quantity > maxQuantity {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}

	values, err := reader(address, quantity)
	if err != nil {
		s.logger.Printf("read bits error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}

	packed := packBools(values)
	resp := make([]byte, 2+len(packed))
	resp[0] = pdu[0]
	resp[1] = byte(len(packed))
	copy(resp[2:], packed)
	return resp
}

func (s *Server) handleReadRegisters(pdu []byte, reader func(uint16, uint16) ([]uint16, error), maxQuantity uint16) []byte {
	address, quantity, ok := parseReadRequest(pdu)
	if !ok || quantity > maxQuantity {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}

	values, err := reader(address, quantity)
	if err != nil {
		s.logger.Printf("read registers error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}

	resp := make([]byte, 2+len(values)*2)
	resp[0] = pdu[0]
	resp[1] = byte(len(values) * 2)
	for i, value := range values {
		binary.BigEndian.PutUint16(resp[2+i*2:], value)
	}
	return resp
}

func (s *Server) handleWriteSingleCoil(pdu []byte) []byte {
	if len(pdu) != 5 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	address := binary.BigEndian.Uint16(pdu[1:3])
	rawValue := binary.BigEndian.Uint16(pdu[3:5])

	var value bool
	switch rawValue {
	case 0xFF00:
		value = true
	case 0x0000:
		value = false
	default:
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}

	if err := s.store.WriteSingleCoil(address, value); err != nil {
		s.logger.Printf("write single coil error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}
	return append([]byte(nil), pdu...)
}

func (s *Server) handleWriteSingleRegister(pdu []byte) []byte {
	if len(pdu) != 5 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	address := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])

	if err := s.store.WriteSingleRegister(address, value); err != nil {
		s.logger.Printf("write single register error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}
	return append([]byte(nil), pdu...)
}

func (s *Server) handleWriteMultipleCoils(pdu []byte) []byte {
	if len(pdu) < 6 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	address := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])

	if quantity == 0 || quantity > 1968 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	expected := 6 + byteCount
	if len(pdu) != expected || byteCount != int((quantity+7)/8) {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}

	values := unpackBools(pdu[6:], int(quantity))
	if err := s.store.WriteMultipleCoils(address, values); err != nil {
		s.logger.Printf("write multiple coils error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}

	resp := make([]byte, 5)
	resp[0] = pdu[0]
	copy(resp[1:], pdu[1:5])
	return resp
}

func (s *Server) handleWriteMultipleRegisters(pdu []byte) []byte {
	if len(pdu) < 6 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	address := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])

	if quantity == 0 || quantity > 123 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}
	expected := 6 + byteCount
	if len(pdu) != expected || byteCount != int(quantity)*2 {
		return exceptionPDU(pdu[0], exceptionIllegalDataValue)
	}

	values := make([]uint16, quantity)
	for i := 0; i < int(quantity); i++ {
		values[i] = binary.BigEndian.Uint16(pdu[6+i*2:])
	}

	if err := s.store.WriteMultipleRegisters(address, values); err != nil {
		s.logger.Printf("write multiple registers error: %v", err)
		return exceptionPDU(pdu[0], exceptionIllegalDataAddress)
	}

	resp := make([]byte, 5)
	resp[0] = pdu[0]
	copy(resp[1:], pdu[1:5])
	return resp
}

func parseReadRequest(pdu []byte) (uint16, uint16, bool) {
	if len(pdu) != 5 {
		return 0, 0, false
	}
	address := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	if quantity == 0 {
		return 0, 0, false
	}
	return address, quantity, true
}

func writeResponse(conn net.Conn, transactionID uint16, unitID uint8, pdu []byte) (int, error) {
	response := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(response[0:2], transactionID)
	binary.BigEndian.PutUint16(response[2:4], 0)
	binary.BigEndian.PutUint16(response[4:6], uint16(len(pdu)+1))
	response[6] = unitID
	copy(response[7:], pdu)
	written, err := conn.Write(response)
	return written, err
}

func exceptionPDU(functionCode uint8, exceptionCode uint8) []byte {
	return []byte{functionCode | 0x80, exceptionCode}
}

func packBools(values []bool) []byte {
	packed := make([]byte, (len(values)+7)/8)
	for i, value := range values {
		if value {
			packed[i/8] |= 1 << uint(i%8)
		}
	}
	return packed
}

func unpackBools(data []byte, quantity int) []bool {
	values := make([]bool, quantity)
	for i := 0; i < quantity; i++ {
		values[i] = data[i/8]&(1<<uint(i%8)) != 0
	}
	return values
}
