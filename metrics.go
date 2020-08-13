package main

import (
	"fmt"
	"math"
	"runtime"
	"time"
)

var (
	epsilon = 32
)

func SetMemUsage(x int, y int, a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) *[][]int8 {
	var overall [][]int8
	var i uint

	mem := getMemoryUsage(x, y, a, b, c, d, e, f, g, h)
	for ; i < mem; i++ {
		a := make([]int8, 0, 1048576*epsilon)
		overall = append(overall, a)
		//memUsage(mem)
		if i % 100 == 0 {
			time.Sleep(time.Millisecond * 10)
		}
	}
	memUsage(mem)
	return &overall
}

func getMemoryUsage(x int, y int, a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) uint {
	var x_param float64
	var y_param float64
	if x%2 == 0 {
		x_param = func_x_1(a, b, c, d)
	} else {
		x_param = func_x_2(d, e, h)
	}
	if y%2 == 0 {
		y_param = func_y_1(a, c, e, f, g)
	} else {
		y_param = func_y_2(b, e, f)
	}
	return himmelblau(x_param, y_param)
}

func FreeMemUsed(overall *[][]int8) {
	overall = nil
	fmt.Printf("free memory")
	memUsage(0)
	runtime.GC()
	memUsage(0)
}

func memUsage(expected uint) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGc = %v\n", m.NumGC)
	fmt.Printf("\tExpected = %v\n", expected)
}

func bToMb(b uint64) uint64 {
	return b / uint64(epsilon) / 1024 / 1024
}

// https://caffinc.github.io/2016/03/cpu-load-generator/
func InfinityCpuUsage(x int, y int, a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) {
	FinityCpuUsage(0, x, y, a, b, c, d, e, f, g, h)
}

func getCpuUsage(x int, y int, a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) float64 {
	var x_param float64
	var y_param float64
	if x%2 == 0 {
		x_param = func_x_1(a, b, c, d)
	} else {
		x_param = func_x_2(d, e, h)
	}
	if y%2 == 0 {
		y_param = func_y_1(a, c, e, f, g)
	} else {
		y_param = func_y_2(b, e, f)
	}
	return beale(x_param, y_param) * 100
}

// set CPU usage in $load% for $timeElapsed ms
func FinityCpuUsage(timeElapsed uint, x int, y int, a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) time.Duration {
	start := time.Now()
	sleepTime := getCpuUsage(x, y, a, b, c, d, e, f, g, h)
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
		fmt.Printf("beale val: %f\n", val)
	} else {
		fmt.Printf("beale fallback 0.4\n")
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
		// Himmelblau Function
		pct = math.Pow(math.Pow(x, 2)+y-11, 2) + math.Pow(x+math.Pow(y, 2)-7, 2)
		// Maximum for this function is approx. 890
		pct = pct / 890 // make it a value between 0 and 1
		fmt.Printf("himmelblau val: %f\n", pct)
	} else {
		fmt.Printf("himmelblau fallback 1\n")
		// For these cases, we received values outside of our valid search domains
		// Therefore, let's just use a catch-all function that'll take any parameters
		// For now, setting it to 100% - can write a function later
		pct = 1
	}
	// this can be changed; let's use 1024 as our maximum and 0 as our minimum
	val := uint(float64(1024) * pct) // note that we're changing it to a uint32 => nearest integer val
	return val
}

func func_x_1(a float64, b float64, c float64, d float64) float64 {
	// Our function won't work if log(d) = 0 because of a divide-by-zero, so just return the maximal value 5
	if math.Log(d) == 0 {
		return 5
	}
	// (a^2 + bc) / (500*log(d))
	return (math.Pow(a, 2) + (b * c)) / (500 * math.Log(d))
}

func func_y_1(a float64, c float64, e float64, f float64, g float64) float64 {
	// sin(a*c*pi) * cos(f^g*pi) - 2*e
	return (math.Sin((a/c)*math.Pi)*math.Cos(f*g*math.Pi) - 2*e)
}

func func_x_2(d float64, e float64, h float64) float64 {
	return (math.Log(d) - (e * h / 32))
}

func func_y_2(b float64, e float64, f float64) float64 {
	if b*e < 0 {
		// Just to make sure that we end up with a positive value to sqrt
		e = -e
	}
	return math.Sqrt(b*e) / f
}
