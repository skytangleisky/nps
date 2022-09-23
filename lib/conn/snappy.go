package conn

import (
	"errors"
	"net"
	"time"

	"github.com/golang/snappy"
)

type SnappyConn struct {
	w *snappy.Writer
	r *snappy.Reader
	c net.Conn
}

func (s *SnappyConn) LocalAddr() net.Addr {
	return s.c.LocalAddr()
}

func (s *SnappyConn) RemoteAddr() net.Addr {
	return s.c.RemoteAddr()
}

func (s *SnappyConn) SetDeadline(t time.Time) error {
	return s.c.SetDeadline(t)
}

func (s *SnappyConn) SetReadDeadline(t time.Time) error {
	return s.c.SetReadDeadline(t)
}

func (s *SnappyConn) SetWriteDeadline(t time.Time) error {
	return s.c.SetWriteDeadline(t)
}

func NewSnappyConn(conn net.Conn) *SnappyConn {
	c := new(SnappyConn)
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	c.c = conn
	return c
}

// snappy压缩写
func (s *SnappyConn) Write(b []byte) (n int, err error) {
	if n, err = s.w.Write(b); err != nil {
		return
	}
	if err = s.w.Flush(); err != nil {
		return
	}
	return
}

// snappy压缩读
func (s *SnappyConn) Read(b []byte) (n int, err error) {
	return s.r.Read(b)
}

func (s *SnappyConn) Close() error {
	err := s.w.Close()
	err2 := s.c.Close()
	if err != nil && err2 == nil {
		return err
	}
	if err == nil && err2 != nil {
		return err2
	}
	if err != nil && err2 != nil {
		return errors.New(err.Error() + err2.Error())
	}
	return nil
}
