# gopool
## feather
1. faster
2. save memory
## usage
```golang
    package main
    
    import (
    	"fmt"
    	"sync/atomic"
    	"time"
    )
    
    func main() {
    
    	pool := NewPool(10000, 10000)
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
    		}
    	}
    	pool.WaitAll()
    
    }

```