package turbotunnel

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
	//	"www.bamsoftware.com/git/turbotunnel-paper.git/example/turbotunnel/turbotunnel"
)

// var errClosed = errors.New("operation on closed connection")
var errNotImplemented = errors.New("not implemented")

// stringAddr satisfies the net.Addr interface using fixed strings for the
// Network and String methods.
type stringAddr struct{ network, address string }

func (addr stringAddr) Network() string { return addr.network }
func (addr stringAddr) String() string  { return addr.address }

// RedialPacketConn implements the net.PacketConn interface by continually
// dialing a static TCP address, and encapsulating packets on each successive
// TCP connection using turbotunnel.ReadPacket and turbotunnel.WritePacket.
//
// Every Turbo Tunnel design will need some sort of PacketConn adapter that
// adapts the session layer's sequence of packets to the obfuscation layer. But
// not every such adapter will look like RedialPacketConn. It depends on what
// the obfuscation layer looks like. Some obfuscation layers will not need a
// persistent connection. One could, for example, handle every ReadFrom or
// WriteTo as an independent network operation.
type RedialPacketConn struct {
	sessionID  SessionID
	remoteAddr net.Addr
	recvQueue  chan []byte
	sendQueue  chan []byte
	closeOnce  sync.Once
	closed     chan struct{}
	ptconn     net.Conn
	// What error to return when the RedialPacketConn is closed.
	err atomic.Value
}

func NewRedialPacketConn(sessionID SessionID, ptconn net.Conn) *RedialPacketConn {
	c := &RedialPacketConn{
		sessionID:  sessionID,
		remoteAddr: ptconn.RemoteAddr(),
		recvQueue:  make(chan []byte, 32),
		sendQueue:  make(chan []byte, 32),
		closed:     make(chan struct{}),
		ptconn:     ptconn,
	}
	go func() {
		c.closeWithError(c.loop())
	}()
	return c
}

// loop dials c.remoteAddr in a loop, exchanging packets on each new connection
// as long as it lasts. Only errors in dialing break the loop and report the
// error to the caller.
func (c *RedialPacketConn) loop() error {
	for {
		select {
		case <-c.closed:
			return nil
		default:
		}
		log.Printf("session %v: redialing %v", c.sessionID, c.remoteAddr)
		err := c.dialAndExchange()
		if err != nil {
			return err
		}
	}
}

func (c *RedialPacketConn) dialAndExchange() error {
	conn := c.ptconn
	defer conn.Close()

	// Begin by sending the session identifier; everything after that is
	// encapsulated packets.
	_, err := conn.Write(c.sessionID[:])
	if err != nil {
		// Errors after the dial are not fatal but cause a redial.
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(2)
	done := make(chan struct{})
	// Read encapsulated packets from the connection and write them to
	// c.recvQueue.
	go func() {
		defer wg.Done()
		defer close(done) // Signal the write loop to finish.
		br := bufio.NewReader(conn)
		for {
			p, err := ReadPacket(br)
			if err != nil {
				return
			}
			select {
			case <-c.closed:
				return
			case c.recvQueue <- p:
			}
		}
	}()
	// Read packets from c.sendQueue and encapsulate them into the
	// connection.
	go func() {
		defer wg.Done()
		defer conn.Close() // Signal the read loop to finish.
		bw := bufio.NewWriter(conn)
		for {
			select {
			case <-c.closed:
				return
			case <-done:
				return
			case p := <-c.sendQueue:
				err := WritePacket(bw, p)
				if err != nil {
					return
				}
				err = bw.Flush()
				if err != nil {
					return
				}
			}
		}
	}()

	// Exchange packets until the connection is terminated.
	wg.Wait()
	return nil
}

func (c *RedialPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case <-c.closed:
		return 0, nil, &net.OpError{Op: "read", Net: c.remoteAddr.Network(), Source: c.sessionID, Addr: c.remoteAddr, Err: c.err.Load().(error)}
	default:
	}
	select {
	case <-c.closed:
		return 0, nil, &net.OpError{Op: "read", Net: c.remoteAddr.Network(), Source: c.sessionID, Addr: c.remoteAddr, Err: c.err.Load().(error)}
	case buf := <-c.recvQueue:
		return copy(p, buf), c.remoteAddr, nil
	}
}

func (c *RedialPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	select {
	case <-c.closed:
		return 0, &net.OpError{Op: "write", Net: c.remoteAddr.Network(), Source: c.sessionID, Addr: c.remoteAddr, Err: c.err.Load().(error)}
	default:
	}
	// Copy the slice so that the caller may reuse p.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case c.sendQueue <- buf:
	default: // Silently drop outgoing packets if the send queue is full.
	}
	return len(buf), nil
}

// closeWithError unblocks pending operations and makes future operations fail
// with the given error. If err is nil, it becomes errClosed.
func (c *RedialPacketConn) closeWithError(err error) error {
	firstClose := false
	c.closeOnce.Do(func() {
		firstClose = true
		// Store the error that will be returned for future operations.
		if err == nil {
			err = errClosed
		}
		c.err.Store(err)
		close(c.closed)
	})
	if !firstClose {
		return &net.OpError{Op: "close", Net: c.remoteAddr.Network(), Source: c.sessionID, Addr: c.remoteAddr, Err: c.err.Load().(error)}
	}
	return nil
}

func (c *RedialPacketConn) Close() error { return c.closeWithError(nil) }

func (c *RedialPacketConn) LocalAddr() net.Addr  { return c.sessionID }
func (c *RedialPacketConn) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *RedialPacketConn) SetDeadline(t time.Time) error      { return errNotImplemented }
func (c *RedialPacketConn) SetReadDeadline(t time.Time) error  { return errNotImplemented }
func (c *RedialPacketConn) SetWriteDeadline(t time.Time) error { return errNotImplemented }
