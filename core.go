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
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/process"

	"github.com/google/uuid"
)

func main() {

	storePid()

	r := mux.NewRouter()
	r.HandleFunc("/cpu", CpuHandler)
	r.HandleFunc("/memory", MemoryHandler)
	r.HandleFunc("/call/{next}", CallNextHandler)
	r.HandleFunc("/serve", RequestHandler)
	r.HandleFunc("/", RequestHandler)
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8888", r))
}

func storePid() {
	pid := os.Getpid()
	bpid := []byte(strconv.Itoa(pid))
	procUUID := uuid.New().String()

	pidFilename := fmt.Sprint("/tmp/", procUUID, ".pid")
	ioutil.WriteFile(pidFilename, bpid, 0644)
}

func CpuHandler(w http.ResponseWriter, r *http.Request) {
	proc := getProcInstance()

	cpuPerc, _ := proc.CPUPercent()

	w.Write([]byte(strconv.FormatFloat(cpuPerc, 'f', 10, 64)))
}

func getProcInstance() *process.Process {
	pid := os.Getpid()
	proc, _ := process.NewProcess(int32(pid))

	return proc
}

func MemoryHandler(w http.ResponseWriter, r *http.Request) {
	proc := getProcInstance()

	memInfo, _ := proc.MemoryInfo()

	w.Write([]byte(strconv.FormatUint(memInfo.VMS, 10)))
}

func CallNextHandler(w http.ResponseWriter, r *http.Request) {

}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	n_cpus := runtime.NumCPU()

	mem := setMemUsage(1024 * 1024 * 1024 * 4)

	load := .8
	for index := 0; index < n_cpus-1; index++ {
		go forceCpuUsage(1000*5, load)
	}

	forceCpuUsage(1000*5, load)
	freeMemUsed(mem)
	// w.Header().Set("Content-Type", "application/text")
	w.Write(make([]byte, 256))
}

func forceCpuUsage(timeElapsed uint, load float64) time.Duration {
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

func setMemUsage(b _Ctype_ulong) *_Ctype_char {
	log.Printf("Allocating %d bytes of memory", b)
	ptr := C.memory_allocation(b)
	return ptr
}

func freeMemUsed(ptr *_Ctype_char) {
	log.Print("Deallocating memory")
	defer C.memory_deallocation(ptr)
	runtime.GC()
	runtime.GC()
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
	return b / 1024 / 1024
}
