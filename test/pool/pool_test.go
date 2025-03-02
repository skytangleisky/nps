package pool

import (
	"ehang.io/nps/lib/common"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_pool(t *testing.T) {
	// 初始化池，最大容量为 100
	pool := common.NewObjectPool(10)
	wg := new(sync.WaitGroup)
	N := 20
	wg.Add(N)
	// 模拟获取和归还对象的操作
	for i := 0; i < N; i++ {
		go func(id int) {
			obj := pool.Get()
			fmt.Printf("Goroutine %d got object,%d\n", id, pool.GetPoolSize())
			// 模拟处理对象
			time.Sleep(time.Millisecond * 1000)
			pool.Put(obj)
			fmt.Printf("Goroutine %d returned object,%d\n", id, pool.GetPoolSize())
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Printf("Final pool size: %d\n", pool.GetPoolSize())
}
