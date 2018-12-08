package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var mimeType = "application/octet-stream"

var (
	name           string
	payload        uint64
	cpuLoad        float64
	memoryRequest  uint64
	processingTime uint
	pidFilepath    string
)

func main() {

	flag.StringVar(&name, "name", uuid.New().String(), "Service name.")
	flag.StringVar(&pidFilepath, "pid-path", "", "Path to save pid. Default (.) .")
	flag.Uint64Var(&payload, "payload", 128, "Average size of payload sent in every messsages (bytes). Default = 128 bytes")
	flag.Uint64Var(&memoryRequest, "memory", 1024, "Average size of memory allocated by each request (bytes). Default = 1024 bytes")
	flag.UintVar(&processingTime, "processing-time", 100, "Average time in each request (ms). Default = 100 ms")
	flag.Float64Var(&cpuLoad, "cpu-load", 0.01, "Average cpu load per request being processed (%). Default = 1%")

	flag.Parse()

	Pid(fmt.Sprint(pidFilepath, name))

	r := mux.NewRouter()
	r.HandleFunc("/cpu", cpuHandler)
	r.HandleFunc("/memory", memoryHandler)
	r.HandleFunc("/processAndCall", processAndCallHandler)
	r.HandleFunc("/callAndProcess", callAndProcessHandler)
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8888", r))
}

func cpuHandler(w http.ResponseWriter, r *http.Request) {
	cpuPerc := GetCpuUsage()
	w.Write([]byte(strconv.FormatFloat(cpuPerc, 'f', 10, 64)))
}

func memoryHandler(w http.ResponseWriter, r *http.Request) {
	memUsage := GetMemUsage()
	w.Write([]byte(strconv.FormatUint(memUsage, 10)))
}

func processAndCallHandler(w http.ResponseWriter, r *http.Request) {
	payloadSize := payload
	destination := ""
	cpuLoad := cpuLoad
	memoryRequest := memoryRequest
	timeToElaspseInMilliseconds := processingTime

	behavior := RequestBeforeProcess{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: timeToElaspseInMilliseconds}}
	result := innerBehavior(behavior, destination, payloadSize)

	w.Header().Set("Content-Type", mimeType)
	w.Write(result)
}

func innerBehavior(behavior InnerBehavior, destination string, payloadSize uint64) []byte {
	result := behavior.Execute(destination, payloadSize)
	return result
}

func callAndProcessHandler(w http.ResponseWriter, r *http.Request) {
	payloadSize := payload
	destination := ""
	cpuLoad := cpuLoad
	memoryRequest := memoryRequest
	timeToElaspseInMilliseconds := processingTime

	behavior := ProcessBeforeRequest{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: timeToElaspseInMilliseconds}}
	result := innerBehavior(behavior, destination, payloadSize)

	w.Header().Set("Content-Type", mimeType)
	w.Write(result)
}
