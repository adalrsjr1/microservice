package main

import (
	"fmt"
	"math"
	"runtime"
	"time"
)

<<<<<<< HEAD
var (
  epsilon = 32
)

func SetMemUsage(b uint) *[][]int8 {
	var overall [][]int8
	var i uint

=======
func SetMemUsage(x float64, y float64) *[][]int8 {
	var overall [][]int8
	var i uint
	b := himmelblau(x, y)
>>>>>>> Add two functions to represent CPU load and memory usage
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
func InfinityCpuUsage(x float64, y float64) {
	FinityCpuUsage(0, x, y)
}

// set CPU usage in $load% for $timeElapsed ms
func FinityCpuUsage(timeElapsed uint, x float64, y float64) time.Duration {
	start := time.Now()
	// beale() will generate a value between 0.2 and 0.4
	sleepTime := beale(x, y) * 100
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

func beale(x float64, y float64) float64 {
	var val = 0.0
	// We use our range as [-4.5, 4.5] because it starts to grow too rapidly beyond those values
	if x >= -4.5 && x <= 4.5 && y >= -4.5 && y <= 4.5 {
		// Beale Function
		val = math.Pow((1.5-x+x*y), 2) + math.Pow((2.25-x+(x*math.Pow(y, 2))), 2) + math.Pow((2.625-x+(x*math.Pow(y, 3))), 2)
		// Maximum for this function is approx. 178000, so we can normalize it to a value between 0 and 0.2 by dividing by 890000
		val = val/890000 + 0.2 // Make it a value between 0.2 and 0.4
	} else {
		// For these cases, we received values outside of our valid search domains
		// Therefore, let's just use a catch-all function that'll take any parameters
		// For now, setting it to our upper bound of 0.4 (40% cpu load) - can write a function later
		val = 0.4
	}
	return val
}

func himmelblau(x float64, y float64) uint {
	var pct = 0.0
	if x >= -5 && x <= 5 && y >= -5 && y <= 5 {
		// Beale Function
		pct = math.Pow(math.Pow(x, 2)+y-11, 2) + math.Pow(x+math.Pow(y, 2)-7, 2)
		// Maximum for this function is approx. 890
		pct = pct / 890 // make it a value between 0 and 1
	} else {
		// For these cases, we received values outside of our valid search domains
		// Therefore, let's just use a catch-all function that'll take any parameters
		// For now, setting it to 100% - can write a function later
		pct = 1
	}
	// this can be changed; let's use 1024 as our maximum and 0 as our minimum
	val := uint(float64(1024) * pct) // note that we're changing it to a uint32 => nearest integer val
	return val
}
