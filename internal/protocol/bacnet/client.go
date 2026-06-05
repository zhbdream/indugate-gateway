package bacnet

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type covHandler func(ObjectRef, any)

type UDPClient struct {
	conn        *net.UDPConn
	target      *net.UDPAddr
	pending     map[byte]chan []byte
	pendingMu   sync.Mutex
	covHandlers map[uint32]covHandler
	covMu       sync.RWMutex
	loopOnce    sync.Once
}

func NewUDPClient(host string, port int) (*UDPClient, error) {
	local, err := net.ResolveUDPAddr("udp4", ":0")
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", local)
	if err != nil {
		return nil, fmt.Errorf("bind udp: %w", err)
	}
	remote, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("resolve target: %w", err)
	}
	return &UDPClient{
		conn:        conn,
		target:      remote,
		pending:     make(map[byte]chan []byte),
		covHandlers: make(map[uint32]covHandler),
	}, nil
}

func (c *UDPClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *UDPClient) ensureLoop() {
	c.loopOnce.Do(func() {
		go c.readLoop()
	})
}

func (c *UDPClient) readLoop() {
	buf := make([]byte, 1500)
	for {
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		data := append([]byte(nil), buf[:n]...)
		if len(data) < 7 {
			continue
		}
		switch data[6] {
		case 0x00:
			if len(data) > 7 {
				c.deliverPending(data[7], data)
			}
		case 0x10:
			if len(data) > 7 && data[7] == 0x02 {
				c.dispatchCOV(data)
			}
		}
	}
}

func (c *UDPClient) registerPending(invokeID byte) chan []byte {
	ch := make(chan []byte, 1)
	c.pendingMu.Lock()
	c.pending[invokeID] = ch
	c.pendingMu.Unlock()
	return ch
}

func (c *UDPClient) unregisterPending(invokeID byte) {
	c.pendingMu.Lock()
	delete(c.pending, invokeID)
	c.pendingMu.Unlock()
}

func (c *UDPClient) deliverPending(invokeID byte, data []byte) {
	c.pendingMu.Lock()
	ch, ok := c.pending[invokeID]
	c.pendingMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- data:
	default:
	}
}

func (c *UDPClient) RegisterCOVHandler(processID uint32, handler covHandler) {
	c.covMu.Lock()
	c.covHandlers[processID] = handler
	c.covMu.Unlock()
}

func (c *UDPClient) UnregisterCOVHandler(processID uint32) {
	c.covMu.Lock()
	delete(c.covHandlers, processID)
	c.covMu.Unlock()
}

func (c *UDPClient) dispatchCOV(data []byte) {
	obj, value, ok := parseCOVNotification(data)
	if !ok {
		return
	}
	processID := uint32(0)
	for i := 6; i < len(data)-1; i++ {
		if data[i] == 0x09 {
			processID = uint32(data[i+1])
			break
		}
	}
	c.covMu.RLock()
	handler := c.covHandlers[processID]
	c.covMu.RUnlock()
	if handler != nil {
		handler(obj, value)
	}
}

func (c *UDPClient) transact(req []byte, invokeID byte, timeout time.Duration) ([]byte, error) {
	c.ensureLoop()
	ch := c.registerPending(invokeID)
	defer c.unregisterPending(invokeID)
	if err := c.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.WriteToUDP(req, c.target); err != nil {
		return nil, fmt.Errorf("send bacnet request: %w", err)
	}
	select {
	case <-time.After(timeout):
		return nil, fmt.Errorf("bacnet request timeout")
	case resp := <-ch:
		return resp, nil
	}
}

func (c *UDPClient) ReadProperty(object ObjectRef, timeout time.Duration, invokeID *uint32) (any, error) {
	id := byte(atomic.AddUint32(invokeID, 1) & 0xff)
	if id == 0 {
		id = 1
	}
	req := buildReadProperty(object, id)
	resp, err := c.transact(req, id, timeout)
	if err != nil {
		return nil, fmt.Errorf("receive read property: %w", err)
	}
	return parsePropertyValue(resp)
}

func (c *UDPClient) WriteProperty(object ObjectRef, value any, timeout time.Duration, invokeID *uint32) error {
	payload, err := encodePresentValue(object, value)
	if err != nil {
		return err
	}
	id := byte(atomic.AddUint32(invokeID, 1) & 0xff)
	if id == 0 {
		id = 1
	}
	req := buildWritePropertyPresentValue(object, id, payload)
	resp, err := c.transact(req, id, timeout)
	if err != nil {
		return fmt.Errorf("receive write property ack: %w", err)
	}
	return parseSimpleACK(resp)
}

func (c *UDPClient) SubscribeCOV(object ObjectRef, processID uint32, lifetime uint32, timeout time.Duration, invokeID *uint32) error {
	id := byte(atomic.AddUint32(invokeID, 1) & 0xff)
	if id == 0 {
		id = 1
	}
	req := buildSubscribeCOV(object, processID, lifetime, id)
	resp, err := c.transact(req, id, timeout)
	if err != nil {
		return err
	}
	return parseSimpleACK(resp)
}

func (c *UDPClient) UnsubscribeCOV(processID uint32, timeout time.Duration, invokeID *uint32) error {
	id := byte(atomic.AddUint32(invokeID, 1) & 0xff)
	if id == 0 {
		id = 1
	}
	req := buildSubscribeCOVCancel(processID, id)
	resp, err := c.transact(req, id, timeout)
	if err != nil {
		return err
	}
	return parseSimpleACK(resp)
}
