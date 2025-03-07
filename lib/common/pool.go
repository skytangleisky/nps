package common

import (
	"sync"
)

const PoolSize = 64 * 1024
const PoolSizeUdp = 1500 - 40 - 8

var BufPoolUdp = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSizeUdp)
	},
}

var BufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSize)
	},
}
