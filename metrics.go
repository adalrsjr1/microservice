package main

/*
#cgo LDFLAGS:
#include<stdlib.h>
#include<string.h>
char* memory_allocation(unsigned long bytes) {
    char *memory = malloc(bytes);
	memset(memory, 1, bytes);
    return memory;
}

void memory_deallocation(char* memory_chunck) {
    return free(memory_chunck);
}
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/process"
)

func ForceCpuUsage(timeElapsed uint, load float64) time.Duration {
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

func GetCpuUsage() float64 {
	proc := getProcInstance()

	cpuPerc, _ := proc.CPUPercent()

	return cpuPerc
}

func getProcInstance() *process.Process {
	pid := os.Getpid()
	proc, _ := process.NewProcess(int32(pid))

	return proc
}

func SetMemUsage(b _Ctype_ulong) *_Ctype_char {
	log.Printf("Allocating %d bytes of memory", b)
	ptr := C.memory_allocation(b)
	return ptr
}

func GetMemUsage() uint64 {
	proc := getProcInstance()
	memInfo, _ := proc.MemoryInfo()
	return memInfo.VMS
}

func FreeMemUsed(ptr *_Ctype_char) {
	log.Print("Deallocating memory")
	defer C.memory_deallocation(ptr)
	runtime.GC()
	runtime.GC()
}

func MemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGc = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func Pid() int {
	pid := os.Getpid()

	bpid := []byte(strconv.Itoa(pid))
	procUUID := uuid.New().String()

	pidFilename := fmt.Sprint("/tmp/", procUUID, ".pid")
	ioutil.WriteFile(pidFilename, bpid, 0644)

	return pid
}
