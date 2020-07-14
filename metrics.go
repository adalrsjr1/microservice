package main

import (
	"fmt"
	"runtime"
	"time"
)

var (
  epsilon = 32
)

func SetMemUsage(b uint) *[][]int8 {
	var overall [][]int8
	var i uint

	for ; i < b; i++ {

		a := make([]int8, 0, 1048576 * epsilon)
		overall = append(overall, a)

		memUsage()
		time.Sleep(time.Millisecond * 10)

	}

	memUsage()
	return &overall
}

func FreeMemUsed(overall *[][]int8) {
	overall = nil
	fmt.Printf("free memory")
	memUsage()
	runtime.GC()
	memUsage()
}

func memUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGc = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / uint64(epsilon) / 1024 / 1024
}

// https://caffinc.github.io/2016/03/cpu-load-generator/
func InfinityCpuUsage(load float64) {
	FinityCpuUsage(0, load)
}

// set CPU usage in $load% for $timeElapsed ms
func FinityCpuUsage(timeElapsed uint, load float64) time.Duration {
	start := time.Now()
	sleepTime := int64(100 * (1 - load))
	var elapsed time.Duration
	for {
		unladenTime := time.Now().UnixNano() / int64(time.Millisecond)
		if unladenTime%100 == 0 {
			time.Sleep(time.Millisecond * time.Duration(sleepTime))
		}
		elapsed = time.Now().Sub(start)
		if timeElapsed > 0 && uint(elapsed/time.Millisecond) >= timeElapsed {
			break
		}
	}
	return elapsed
}
