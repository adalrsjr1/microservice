package main

import (
	"bytes"
	"log"
	"net/http"
	"runtime"
	"strconv"

	"github.com/gorilla/http"
	"github.com/gorilla/mux"
)

func main() {

	Pid()

	// var (
	// 	name       string
	// 	msgSize    uint
	// 	load       float64
	// 	msgTime    uint
	// 	memUsage   uint
	// 	zipkinAddr string
	// 	isSampling bool
	// )

	// flag.StringVar(&zipkinAddr, "zipkin", "0.0.0.0:9411", "zipkin addr:port default 0.0.0.0:9411")
	// flag.StringVar(&name, "name", "", "service name")
	// flag.UintVar(&msgSize, "msg-size", 256, "average size in bytes default:256")
	// flag.Float64Var(&load, "load", 0.1, "CPU load per message default:10% (0.1)")
	// flag.UintVar(&msgTime, "msg-time", 10, "Time do compute an msg-request default 10ms")
	// flag.UintVar(&memUsage, "mem", 128, "min memory usage default:128MB")
	// flag.BoolVar(&isSampling, "sampling", true, "sampling messages to store into zipkin")
	// flag.Parse()
	// addrs := flag.Args()

	// 	if len(name) <= 0 {
	// 		log.Fatal("argument --name must be set")
	// 	}

	r := mux.NewRouter()
	r.HandleFunc("/cpu", CpuHandler)
	r.HandleFunc("/memory", MemoryHandler)
	r.HandleFunc("/call/{next}", IntermediaryRequestHandler)
	r.HandleFunc("/", LeafRequestHandler)
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8888", r))
}

func CpuHandler(w http.ResponseWriter, r *http.Request) {
	cpuPerc := GetCpuUsage()
	w.Write([]byte(strconv.FormatFloat(cpuPerc, 'f', 10, 64)))
}

func MemoryHandler(w http.ResponseWriter, r *http.Request) {
	memUsage := GetMemUsage()
	w.Write([]byte(strconv.FormatUint(memUsage, 10)))
}

func IntermediaryRequestHandler(w http.ResponseWriter, r *http.Request) {
	payload := 128
	destination := ""
	cpuLoad := .3
	memoryRequest := 128
	innerBehavior(cpuLoad, memoryRequest, payload, destination, callNext)
}

type fn func(uint, string)

func callNext(payloadSize uint, destination string) {
	if err := gorilla / http.Post(destination, bytes.NewReader(make([]byte, payloadSize))); err != nil {
		log.Fatalf("could not post: %v", err)
	}
}

func innerBehavior(cpuLoad float64, memoryRequest uint64, payloadSize uint, destination string, next fn) {
	next(payloadSize, destination)
}

func intermediaryRequest(r *http.Request) *http.Request {
	form := r.Form
	form.Add("cpu_load", "")
	form.Add("memory_request", "")
	form.Add("payload", "")

	r.Form = form
	return r
}

func LeafRequestHandler(w http.ResponseWriter, r *http.Request) {
	n_cpus := runtime.NumCPU()

	mem := SetMemUsage(128)

	load := .1
	for index := 0; index < n_cpus-1; index++ {
		go ForceCpuUsage(1000*5, load)
	}

	ForceCpuUsage(1000*5, load)
	FreeMemUsed(mem)
	w.Header().Set("Content-Type", "text/plain")
	w.Write(make([]byte, 256))
}
