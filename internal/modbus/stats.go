package modbus

import (
	"sync"
	"time"
)

type ConnectionStats struct {
	RemoteAddr   string
	ConnectedAt  time.Time
	LastSeenAt   time.Time
	BytesIn      uint64
	BytesOut     uint64
	MessagesIn   uint64
	MessagesOut  uint64
	LastUnitID   uint8
	LastFunction uint8
}

type TrafficSnapshot struct {
	StartedAt         time.Time
	ActiveClients     int
	TotalBytesIn      uint64
	TotalBytesOut     uint64
	TotalMessagesIn   uint64
	TotalMessagesOut  uint64
	BytesInPerSecond  float64
	BytesOutPerSecond float64
	MessagesPerSecond float64
	Connections       []ConnectionStats
}

type trafficTotals struct {
	bytesIn     uint64
	bytesOut    uint64
	messagesIn  uint64
	messagesOut uint64
}

type trafficCollector struct {
	mu              sync.Mutex
	startedAt       time.Time
	windowStartedAt time.Time
	window          trafficTotals
	totals          trafficTotals
	connections     map[string]*ConnectionStats
}

func newTrafficCollector(now time.Time) *trafficCollector {
	return &trafficCollector{
		startedAt:       now,
		windowStartedAt: now,
		connections:     make(map[string]*ConnectionStats),
	}
}

func (c *trafficCollector) connectionOpened(remote string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connections[remote] = &ConnectionStats{
		RemoteAddr:  remote,
		ConnectedAt: now,
		LastSeenAt:  now,
	}
}

func (c *trafficCollector) connectionClosed(remote string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.connections, remote)
}

func (c *trafficCollector) recordRead(remote string, bytes int, unitID uint8, functionCode uint8, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totals.bytesIn += uint64(bytes)
	c.totals.messagesIn++
	c.window.bytesIn += uint64(bytes)
	c.window.messagesIn++

	if conn := c.connections[remote]; conn != nil {
		conn.BytesIn += uint64(bytes)
		conn.MessagesIn++
		conn.LastSeenAt = now
		conn.LastUnitID = unitID
		conn.LastFunction = functionCode
	}
}

func (c *trafficCollector) recordWrite(remote string, bytes int, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totals.bytesOut += uint64(bytes)
	c.totals.messagesOut++
	c.window.bytesOut += uint64(bytes)
	c.window.messagesOut++

	if conn := c.connections[remote]; conn != nil {
		conn.BytesOut += uint64(bytes)
		conn.MessagesOut++
		conn.LastSeenAt = now
	}
}

func (c *trafficCollector) snapshot(activeClients int, now time.Time) TrafficSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := now.Sub(c.windowStartedAt).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	connections := make([]ConnectionStats, 0, len(c.connections))
	for _, conn := range c.connections {
		connections = append(connections, *conn)
	}

	snapshot := TrafficSnapshot{
		StartedAt:         c.startedAt,
		ActiveClients:     activeClients,
		TotalBytesIn:      c.totals.bytesIn,
		TotalBytesOut:     c.totals.bytesOut,
		TotalMessagesIn:   c.totals.messagesIn,
		TotalMessagesOut:  c.totals.messagesOut,
		BytesInPerSecond:  float64(c.window.bytesIn) / elapsed,
		BytesOutPerSecond: float64(c.window.bytesOut) / elapsed,
		MessagesPerSecond: float64(c.window.messagesIn) / elapsed,
		Connections:       connections,
	}

	if now.Sub(c.windowStartedAt) >= time.Second {
		c.windowStartedAt = now
		c.window = trafficTotals{}
	}

	return snapshot
}
