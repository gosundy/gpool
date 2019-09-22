package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func main() {

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

	jobCount := 10000000000
	pool.WaitCount(jobCount)
	for i := 0; i < jobCount; i++ {
		k := i
		pool.JobQueue <- func() {
			defer pool.JobDone()
			atomic.AddInt64(&count, int64(k))
		}
	}
	pool.WaitAll()

}
