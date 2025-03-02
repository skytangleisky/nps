package common

import (
	"sync"
	"sync/atomic"
)

type ObjectPool struct {
	pool     sync.Pool
	poolSize int32
	maxSize  int32
	mu       sync.Mutex
	cond     *sync.Cond
}

func NewObjectPool(maxSize int32) *ObjectPool {
	op := &ObjectPool{
		maxSize: maxSize,
	}
	op.cond = sync.NewCond(&op.mu)
	op.pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 65536)
		},
	}
	return op
}

func (op *ObjectPool) Get() interface{} {
	op.mu.Lock()
	defer op.mu.Unlock()

	// 如果池中对象超过最大大小，阻塞直到有空间
	for op.poolSize >= op.maxSize {
		op.cond.Wait() // 阻塞当前 goroutine，等待条件满足
	}

	// 获取对象
	obj := op.pool.Get()

	// 增加池中的对象计数
	atomic.AddInt32(&op.poolSize, 1)

	return obj
}

func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	defer op.mu.Unlock()

	// 归还对象前，减少池中的对象计数
	atomic.AddInt32(&op.poolSize, -1)

	// 归还到池中
	op.pool.Put(obj)

	// 通知等待的 goroutine，有空间可以获取对象
	op.cond.Signal()
}

func (op *ObjectPool) GetPoolSize() int32 {
	return atomic.LoadInt32(&op.poolSize)
}

//var BufPool = NewObjectPool(100)
