package common

import (
	"fmt"
	"math"
	"sync"
)

const PoolSize = 64 * 1024
const PoolSizeSmall = 100
const PoolSizeUdp = 1472 + 200
const PoolSizeCopy = 32 << 10

var BufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSize)
	},
}

var BufPoolUdp = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSizeUdp)
	},
}
var BufPoolMax = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSize)
	},
}
var BufPoolSmall = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSizeSmall)
	},
}
var BufPoolCopy = sync.Pool{
	New: func() interface{} {
		return make([]byte, PoolSizeCopy)
	},
}

func PutBufPoolUdp(buf []byte) {
	if cap(buf) == PoolSizeUdp {
		BufPoolUdp.Put(buf[:PoolSizeUdp])
	}
}

func PutBufPoolCopy(buf []byte) {
	if cap(buf) == PoolSizeCopy {
		BufPoolCopy.Put(buf[:PoolSizeCopy])
	}
}

func GetBufPoolCopy() []byte {
	return (BufPoolCopy.Get().([]byte))[:PoolSizeCopy]
}

func PutBufPoolMax(buf []byte) {
	if cap(buf) == PoolSize {
		BufPoolMax.Put(buf[:PoolSize])
	}
}

type copyBufferPool struct {
	pool sync.Pool
}

func (Self *copyBufferPool) New() {
	Self.pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, PoolSizeCopy, PoolSizeCopy)
		},
	}
}

func (Self *copyBufferPool) Get() []byte {
	buf := Self.pool.Get().([]byte)
	return buf[:PoolSizeCopy] // just like make a new slice, but data may not be 0
}

func (Self *copyBufferPool) Put(x []byte) {
	if len(x) == PoolSizeCopy {
		Self.pool.Put(x)
	} else {
		x = nil // buf is not full, not allowed, New method returns a full buf
	}
}

var once = sync.Once{}
var CopyBuff = copyBufferPool{}

func newPool() {
	CopyBuff.New()
}

func init() {
	once.Do(newPool)
}
func Changeunit(len int64) string {
	//1 Byte(B) = 8bit = 8b
	//1 Kilo    Byte(KB) = 1024B
	//1 Mega    Byte(MB) = 1024KB
	//1 Giga    Byte(GB) = 1024MB
	//1 Tera    Byte(TB) = 1024GB
	//1 Peta    Byte(PB) = 1024TB
	//1 Exa     Byte(EB) = 1024PB
	//1 Zetta   Byte(ZB) = 1024EB
	//1 Yotta   Byte(YB) = 1024ZB
	//1 Bronto  Byte(BB) = 1024YB
	//1 Nona    Byte(NB) = 1024BB
	//1 Dogga   Byte(DB) = 1024NB
	//1 Corydon Byte(CB) = 1024DB
	//1 Xero    Byte(XB) = 1024CB

	var Bit = float64(len)
	var KB = Bit / 1024
	var MB = KB / 1024
	var GB = MB / 1024
	var TB = GB / 1024
	var PB = TB / 1024
	var EB = PB / 1024
	var ZB = EB / 1024
	var YB = ZB / 1024
	var BB = YB / 1024
	var NB = BB / 1024
	var CB = NB / 1024
	var XB = CB / 1024
	if Bit < 1024 {
		return fmt.Sprintf("%.0f", math.Floor(Bit*100)/100) + "B"
	} else if KB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(KB*100)/100) + "KB"
	} else if MB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(MB*100)/100) + "MB"
	} else if GB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(GB*100)/100) + "GB"
	} else if TB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(TB*100)/100) + "TB"
	} else if PB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(PB*100)/100) + "PB"
	} else if EB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(EB*100)/100) + "EB"
	} else if ZB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(ZB*100)/100) + "ZB"
	} else if YB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(YB*100)/100) + "YB"
	} else if BB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(BB*100)/100) + "BB"
	} else if NB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(NB*100)/100) + "NB"
	} else if CB < 1024 {
		return fmt.Sprintf("%.2f", math.Floor(CB*100)/100) + "CB"
	} else {
		return fmt.Sprintf("%.2f", math.Floor(XB*100)/100) + "XB"
	}
}
