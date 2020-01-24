package vhost

import (
	"bytes"
	"io"
	"net"
	"sync"
)

const (
	initVhostBufSize = 1024 // allocate 1 KB up front to try to avoid resizing
)

type SharedConn struct {
	sync.Mutex
	net.Conn               // the raw connection
	VhostBuf *bytes.Buffer // all of the initial data that has to be read in order to vhost a connection is saved here
}

func newShared(conn net.Conn) (*SharedConn, io.Reader) {
	c := &SharedConn{
		Conn:     conn,
		VhostBuf: bytes.NewBuffer(make([]byte, 0, initVhostBufSize)),
	}

	return c, io.TeeReader(conn, c.VhostBuf)
}

func (c *SharedConn) Read(p []byte) (n int, err error) {
	c.Lock()
	if c.VhostBuf == nil {
		c.Unlock()
		return c.Conn.Read(p)
	}
	n, err = c.VhostBuf.Read(p)

	// end of the request buffer
	if err == io.EOF {
		// let the request buffer Get garbage collected
		// and make sure we don't read from it again
		c.VhostBuf = nil

		// continue reading from the connection
		var n2 int
		n2, err = c.Conn.Read(p[n:])

		// update total read
		n += n2
	}
	c.Unlock()
	return
}
