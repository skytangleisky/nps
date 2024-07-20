package rate

import (
	"sync"
	"sync/atomic"
	"time"
)

type Rate struct {
	count             int64
	bucketSize        int64
	bucketSurplusSize int64
	mu                sync.Mutex
	stopChan          chan bool
	NowRate           int64
}

func NewRate(size int64) *Rate {
	s := &Rate{
		count:             0,
		bucketSize:        size,
		bucketSurplusSize: 0,
		stopChan:          make(chan bool),
	}
	go s.session()
	return s
}

// 停止
func (s *Rate) Stop() {
	s.stopChan <- true
}
func (s *Rate) SetRate(size int64) {
	s.bucketSize = size
	s.bucketSurplusSize = s.bucketSize
}
func (s *Rate) Get(size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bucketSize > 0 {
		if s.bucketSurplusSize >= size {
			atomic.AddInt64(&s.bucketSurplusSize, -size)
			s.count += size
		} else {
			ticker := time.NewTicker(time.Millisecond * 100)
			for {
				select {
				case <-ticker.C:
					if s.bucketSurplusSize >= size {
						atomic.AddInt64(&s.bucketSurplusSize, -size)
						s.count += size
						ticker.Stop()
						return
					}
				case <-s.stopChan:
					ticker.Stop()
					return
				}
			}
		}
	} else {
		s.count += size
	}
}

func (s *Rate) session() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			s.NowRate = s.count
			s.count = 0
			s.bucketSurplusSize = s.bucketSize
		case <-s.stopChan:
			ticker.Stop()
			return
		}
	}
}
