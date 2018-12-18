package main

import (
	"flag"
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
	children       []string
	addr           string
)

var behaviorManager *BehaviorManager

func main() {

	flag.StringVar(&name, "name", uuid.New().String(), "Service name.")
	flag.StringVar(&pidFilepath, "pid-path", "", "Path to save pid. Default (.) .")
	flag.Uint64Var(&payload, "payload", 128, "Average size of payload sent in every messsages (bytes). Default = 128 bytes")
	flag.Uint64Var(&memoryRequest, "memory", 1024, "Average size of memory allocated by each request (bytes). Default = 1024 bytes")
	flag.UintVar(&processingTime, "processing-time", 100, "Average time in each request (ms). Default = 100 ms")
	flag.Float64Var(&cpuLoad, "cpu-load", 0.01, "Average cpu load per request being processed (%). Default = 1%")
	flag.StringVar(&addr, "addr", ":8888", "Addr being used - To use a port value between 8000-8999. Default = :8888")

	flag.Parse()

	children = flag.Args()

	Pid(pidFilepath + name)

	externalBehavior := NewExternalBehavior()
	behaviorManager = NewBehaviorManager(externalBehavior)

	r := mux.NewRouter()

	r.Methods("Post", "Get").Path("/cpu").HandlerFunc(cpuHandler)
	r.Methods("Post", "Get").Path("/memory").HandlerFunc(memoryHandler)
	r.Methods("Post", "Get").Path("/roundrobin/processAndCall").HandlerFunc(roundRobinProcessAndCall)
	r.Methods("Post", "Get").Path("/roundrobin/callAndProcess").HandlerFunc(roundRobinCallAndProcess)
	r.Methods("Post", "Get").Path("/random/processAndCall").HandlerFunc(randomProcessAndCall)
	r.Methods("Post", "Get").Path("/random/callAndProcess").HandlerFunc(randomCallAndProcess)
	r.Methods("Post", "Get").Path("/all/processAndCall").HandlerFunc(allProcessAndCall)
	r.Methods("Post", "Get").Path("/all/callAndProcess").HandlerFunc(allCallAndProcess)

	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(addr, r))
}

func cpuHandler(w http.ResponseWriter, r *http.Request) {
	cpuPerc := GetCpuUsage()
	w.Write([]byte(strconv.FormatFloat(cpuPerc, 'f', 10, 64)))
}

func memoryHandler(w http.ResponseWriter, r *http.Request) {
	memUsage := GetMemUsage()
	w.Write([]byte(strconv.FormatUint(memUsage, 10)))
}

type BehaviorManager struct {
	externalBehavior *ExternalBehavior
}

func NewBehaviorManager(externalBehavior *ExternalBehavior) *BehaviorManager {
	manager := new(BehaviorManager)
	manager.externalBehavior = externalBehavior

	return manager
}

func (b BehaviorManager) roundRobin(children []string, innerBehavior InnerBehavior, w http.ResponseWriter) []byte {
	result := b.externalBehavior.RoundRobin(children, payload, innerBehavior)
	w.Header().Set("Content-Type", mimeType)
	w.Write(result)
	return result
}

func (b BehaviorManager) random(children []string, innerBehavior InnerBehavior, w http.ResponseWriter) []byte {
	result := b.externalBehavior.Random(children, payload, innerBehavior)
	w.Header().Set("Content-Type", mimeType)
	w.Write(result)
	return result
}

func (b BehaviorManager) all(children []string, innerBehavior InnerBehavior, w http.ResponseWriter) []byte {
	result := b.externalBehavior.All(children, payload, innerBehavior)
	w.Header().Set("Content-Type", mimeType)
	w.Write(result)
	return result
}

func roundRobinProcessAndCall(w http.ResponseWriter, r *http.Request) {
	behavior := ProcessBeforeRequest{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.roundRobin(children, behavior, w)
}

func roundRobinCallAndProcess(w http.ResponseWriter, r *http.Request) {
	behavior := RequestBeforeProcess{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.roundRobin(children, behavior, w)
}

func randomProcessAndCall(w http.ResponseWriter, r *http.Request) {
	behavior := ProcessBeforeRequest{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.random(children, behavior, w)
}

func randomCallAndProcess(w http.ResponseWriter, r *http.Request) {
	behavior := RequestBeforeProcess{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.random(children, behavior, w)
}

func allProcessAndCall(w http.ResponseWriter, r *http.Request) {
	behavior := ProcessBeforeRequest{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.all(children, behavior, w)
}

func allCallAndProcess(w http.ResponseWriter, r *http.Request) {
	behavior := RequestBeforeProcess{InnerBehaviorBase{cpuLoad: cpuLoad, memoryRequest: memoryRequest, timeToElaspseInMilliseconds: processingTime}}
	behaviorManager.all(children, behavior, w)
}
