package turbotunnel

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var errClosed = errors.New("operation on closed connection")

//var errNotImplemented = errors.New("not implemented")

// taggedPacket is a combination of a []byte and a net.Addr, encapsulating the
// return type of PacketConn.ReadFrom.
type taggedPacket struct {
	P    []byte
	Addr net.Addr
}

// QueuePacketConn implements net.PacketConn by storing queues of packets. There
// is one incoming queue (where packets are additionally tagged by the source
// address of the peer that sent them). There are many outgoing queues, one for
// each remote peer address that has been recently seen. The QueueIncoming
// method inserts a packet into the incoming queue, to eventually be returned by
// ReadFrom. WriteTo inserts a packet into an address-specific outgoing queue,
// which can later by accessed through the OutgoingQueue method.
type QueuePacketConn struct {
	remotes   *RemoteMap
	localAddr net.Addr
	recvQueue chan taggedPacket
	closeOnce sync.Once
	closed    chan struct{}
	// What error to return when the QueuePacketConn is closed.
	err atomic.Value
}

// NewQueuePacketConn makes a new QueuePacketConn, set to track recent peers
// for at least a duration of timeout.
func NewQueuePacketConn(localAddr net.Addr, timeout time.Duration) *QueuePacketConn {
	return &QueuePacketConn{
		remotes:   NewRemoteMap(timeout),
		localAddr: localAddr,
		recvQueue: make(chan taggedPacket, 32),
		closed:    make(chan struct{}),
	}
}

// QueueIncoming queues an incoming packet and its source address, to be
// returned in a future call to ReadFrom.
func (c *QueuePacketConn) QueueIncoming(p []byte, addr net.Addr) {
	select {
	case <-c.closed:
		// If we're closed, silently drop it.
		return
	default:
	}
	// Copy the slice so that the caller may reuse p.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case c.recvQueue <- taggedPacket{buf, addr}:
	default: // Silently drop incoming packets if the receive queue is full.
	}
}

// OutgoingQueue returns the queue of outgoing packets corresponding to addr,
// creating it if necessary. The contents of the queue will be packets that are
// written to the address in question using WriteTo.
func (c *QueuePacketConn) OutgoingQueue(addr net.Addr) <-chan []byte {
	return c.remotes.SendQueue(addr)
}

// ReadFrom returns a packet and address previously stored by QueueIncoming.
func (c *QueuePacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case <-c.closed:
		return 0, nil, &net.OpError{Op: "read", Net: c.LocalAddr().Network(), Addr: c.LocalAddr(), Err: c.err.Load().(error)}
	default:
	}
	select {
	case <-c.closed:
		return 0, nil, &net.OpError{Op: "read", Net: c.LocalAddr().Network(), Addr: c.LocalAddr(), Err: c.err.Load().(error)}
	case packet := <-c.recvQueue:
		return copy(p, packet.P), packet.Addr, nil
	}
}

// WriteTo queues an outgoing packet for the given address. The queue can later
// be retrieved using the OutgoingQueue method.
func (c *QueuePacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	select {
	case <-c.closed:
		return 0, &net.OpError{Op: "write", Net: c.LocalAddr().Network(), Addr: c.LocalAddr(), Err: c.err.Load().(error)}
	default:
	}
	// Copy the slice so that the caller may reuse p.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case c.remotes.SendQueue(addr) <- buf:
	default: // Silently drop outgoing packets if the send queue is full.
	}
	return len(buf), nil
}

// closeWithError unblocks pending operations and makes future operations fail
// with the given error. If err is nil, it becomes errClosed.
func (c *QueuePacketConn) closeWithError(err error) error {
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
		return &net.OpError{Op: "close", Net: c.LocalAddr().Network(), Addr: c.LocalAddr(), Err: c.err.Load().(error)}
	}
	return nil
}

// Close unblocks pending operations and makes future operations fail with a
// "closed connection" error.
func (c *QueuePacketConn) Close() error { return c.closeWithError(nil) }

// LocalAddr returns the localAddr value that was passed to NewQueuePacketConn.
func (c *QueuePacketConn) LocalAddr() net.Addr { return c.localAddr }

func (c *QueuePacketConn) SetDeadline(t time.Time) error      { return errNotImplemented }
func (c *QueuePacketConn) SetReadDeadline(t time.Time) error  { return errNotImplemented }
func (c *QueuePacketConn) SetWriteDeadline(t time.Time) error { return errNotImplemented }
