package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func main() {
	//trace.Start(os.Stderr)
	//defer trace.Stop()
	pool := NewPool(100000, 100000)
	count := int64(0)
	go func() {
		for {
			fmt.Println(pool.running())
			time.Sleep(time.Second)
		}

	}()
	start := time.Now()
	defer func() {

		defer pool.Release()
		fmt.Println(time.Now().Sub(start))
		fmt.Println("res:", count)
		return
	}()

	
	jobCount := 100000000
	pool.WaitCount(jobCount)
	for i := 0; i < jobCount; i++ {
		k := i
		pool.JobQueue <- func() {
			defer pool.JobDone()
			atomic.AddInt64(&count, int64(k))
			time.Sleep(time.Microsecond*1000)
		}
	}
	pool.WaitAll()

}
