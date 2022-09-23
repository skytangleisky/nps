package rate

import (
	"net"
)

type rateConn struct {
	net.Conn
	rate *Rate
}

func NewRateConn(conn net.Conn, rate *Rate) *rateConn {
	tmp := &rateConn{
		Conn: conn,
		rate: rate,
	}

	return tmp
}

func (s *rateConn) Read(b []byte) (n int, err error) {
	n, err = s.Conn.Read(b)
	if s.rate != nil {
		s.rate.Get(int64(n))
	}
	return
}

func (s *rateConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if s.rate != nil {
		s.rate.Get(int64(n))
	}
	return
}

func (s *rateConn) Close() error {
	return s.Conn.Close()
}
