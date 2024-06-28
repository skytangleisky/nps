package common

import (
	"sync"
)

const PoolSize = 512 << 10
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
