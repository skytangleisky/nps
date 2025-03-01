package Debug

import "sync"
import "github.com/gorilla/websocket"

type SafeWebSocketSlice struct {
	mu    sync.RWMutex
	conns []*websocket.Conn
}

func (s *SafeWebSocketSlice) Add(conn *websocket.Conn) {
	s.mu.Lock() // 写锁，确保其他操作不能同时修改切片
	defer s.mu.Unlock()

	s.conns = append(s.conns, conn)
}

func (s *SafeWebSocketSlice) Remove(conn *websocket.Conn) {
	s.mu.Lock() // 写锁
	defer s.mu.Unlock()

	for i, c := range s.conns {
		if c == conn {
			// 删除元素，保持切片连续
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			break
		}
	}
}

func (s *SafeWebSocketSlice) GetAll() []*websocket.Conn {
	s.mu.RLock() // 读锁，允许多个读取操作
	defer s.mu.RUnlock()

	// 返回切片的副本，避免外部修改原始切片
	copyConns := make([]*websocket.Conn, len(s.conns))
	copy(copyConns, s.conns)
	return copyConns
}
