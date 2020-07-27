package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	zipkin "github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
)

var globalName = ""
var tracer *zipkin.Tracer

func main() {
	writePid()
	var (
		name       string
		msgSize    uint
		load       float64
		msgTime    uint
		memUsage   uint
		zipkinAddr string
		isSampling bool
	)

	flag.StringVar(&zipkinAddr, "zipkin", "0.0.0.0:9411", "zipkin addr:port default 0.0.0.0:9411")
	flag.StringVar(&name, "name", "", "service name")
	flag.UintVar(&msgSize, "msg-size", 256, "average size in bytes default:256")
	flag.Float64Var(&load, "load", 0.1, "CPU load per message default:10% (0.1)")
	flag.UintVar(&msgTime, "msg-time", 10, "Time do compute an msg-request default 10ms")
	flag.UintVar(&memUsage, "mem", 128, "min memory usage default:128MB")
	flag.BoolVar(&isSampling, "sampling", true, "sampling messages to store into zipkin")
	flag.Parse()
	addrs := flag.Args()

	if len(name) <= 0 {
		log.Fatal("argument --name must be set")
	}
	globalName = name

	var err error
	tracer, err = newTracer(name, zipkinAddr, isSampling)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", name)

	// create global zipkin traced http client
	client, err := zipkinhttp.NewClient(tracer, zipkinhttp.ClientTrace(true))
	if err != nil {
		log.Fatalf("unable to create client: %+v\n", err)
	}

	// We add the instrumented transport to the defaultClient that comes
	// with the zipkin-go library
	http.DefaultClient.Transport, err = zipkinhttp.NewTransport(
		tracer,
		zipkinhttp.TransportTrace(true),
	)

	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(42)

	SetMemUsage(memUsage)

	r := mux.NewRouter()

	if len(addrs) != 0 {

		log.Printf("setting non-terminal handlers in %s ___________________", name)

		r.Methods("POST").Path("/").HandlerFunc(AllTargets(client, addrs[:], msgSize, msgTime, load))
		r.Methods("POST").Path("/random").HandlerFunc(RandomTarget(client, addrs[:], msgSize, msgTime, load))

		r.Methods("POST").Path("/0").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "0"))
		r.Methods("POST").Path("/1").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "1"))
		r.Methods("POST").Path("/2").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "2"))
		r.Methods("POST").Path("/3").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "3"))

	} else {

		log.Printf("setting terminal handlers in %s", name)

		r.Methods("POST").Path("/").HandlerFunc(TerminationRequest(client, name, addrs[:], msgSize, msgTime, load))
		r.Methods("POST").Path("/random").HandlerFunc(TerminationRequest(client, name, addrs[:], msgSize, msgTime, load))

		r.Methods("POST").Path("/0").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "0"))
		r.Methods("POST").Path("/1").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "1"))
		r.Methods("POST").Path("/2").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "2"))
		r.Methods("POST").Path("/3").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, "3"))

	}

	r.Use(zipkinhttp.NewServerMiddleware(
		tracer,
		zipkinhttp.TagResponseSize(true),
		zipkinhttp.SpanName("request")), // name for request span
	)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func writePid() {
	pid := os.Getpid()
	bpid := []byte(strconv.Itoa(pid))
	ioutil.WriteFile("/tmp/go.pid", bpid, 0644)
}

func getNextTarget(currentNode string, requestType string) string {

	// var routeMap = map[string]map[string]string{
	// 	"0": {
	// 		"svc_0": "svc_2",
	// 		"svc_2": "svc_1",
	// 		"svc_1": "svc_6",
	// 		"svc_6": ""},
	// 	"1": {
	// 		"svc_0": "svc_2",
	// 		"svc_2": "svc_1",
	// 		"svc_1": "svc_4",
	// 		"svc_4": "svc_5",
	// 		"svc_5": ""},
	// 	"2": {
	// 		"svc_0": "svc_2",
	// 		"svc_2": "svc_1",
	// 		"svc_1": "svc_4",
	// 		"svc_4": "svc_3",
	// 		"svc_3": "svc_9",
	// 		"svc_9": ""},
	// 	"3": {
	// 		"svc_0": "svc_2",
	// 		"svc_2": "svc_1",
	// 		"svc_1": "svc_4",
	// 		"svc_4": "svc_3",
	// 		"svc_3": "svc_8",
	// 		"svc_8": "svc_7",
	// 		"svc_7": ""}}

	nextNode := generatedRouteMap[requestType][currentNode]

	return nextNode
}

func handleRequest(name string, client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64, requestType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ok")
		traverseTargets(name, w, r, client, targets[:], requestSize, calculationTime, load, requestType)
	}
}

// Traverse the path defined for that requestType in the generatedRouteMap
func traverseTargets(name string, w http.ResponseWriter, r *http.Request,
	client *zipkinhttp.Client, targets []string, requestSize uint,
	calculationTime uint, load float64, requestType string) {

	span, ctx := tracer.StartSpanFromContext(r.Context(), "size")

	if span == nil {
		log.Printf("span is nil")
	}

	span.Tag("termination_node", "false")

	FinityCpuUsage(calculationTime, load)
	span.Annotate(time.Now(), "expensive_calc_done")

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	target := getNextTarget(name, requestType)

	if target != "" {
		log.Printf("-->" + target)
		newRequest, err := http.NewRequest("POST", "http://"+target+":8080/"+requestType, bytes.NewBuffer(byteMessage))
		span.Tag("source", globalName)
		span.Tag("target", target)
		span.Tag("req-size", strconv.Itoa(binary.Size(byteMessage)))

		if err != nil {
			log.Printf("unable to create client: %+v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		defer span.Finish()

		byteMessage = nil

		ctx = zipkin.NewContext(newRequest.Context(), span)
		newRequest = newRequest.WithContext(ctx)

		res, err := client.DoWithAppSpan(newRequest, "random "+target)
		if err != nil {
			log.Printf("call to %s returned error: %+v\n", target, err)
			http.Error(w, err.Error(), 500)
			return
		}
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		span.Tag("res-size", strconv.Itoa(binary.Size(bodyBytes)))
		res.Body.Close()
		w.Write(bodyBytes)
	} else {
		log.Printf("termination")

		byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))
		w.Header().Set("Content-Type", "octect-stream")
		w.Write(byteMessage)

		span, _ := tracer.StartSpanFromContext(r.Context(), "size")
		span.Tag("termination_node", "true")
		timeElapsed := FinityCpuUsage(calculationTime, load)
		span.Tag("elapsed_time_ms", strconv.FormatInt(int64(timeElapsed/time.Millisecond), 10))

		span.Finish()
	}
}

func AllTargets(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ok")
		CallAllTargets(w, r, client, targets[:], requestSize, calculationTime, load)
	}
}

func CallAllTargets(w http.ResponseWriter, r *http.Request, client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64) {
	FinityCpuUsage(calculationTime, load)
	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	var bodyBytes []byte
	for _, target := range targets {
		span, ctx := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
		span.Tag("termination_node", "false")
		log.Printf("-->" + target)
		newRequest, err := http.NewRequest("POST", "http://"+target+":8080/", bytes.NewBuffer(byteMessage))
		span.Tag("source", globalName)
		span.Tag("target", target)
		span.Tag("req-size", strconv.Itoa(binary.Size(bodyBytes)))

		if err != nil {
			log.Printf("unable to create client: %+v\n", err)
			http.Error(w, err.Error(), 500)
			continue
		}
		defer span.Finish()
		span.Tag("req-size", strconv.Itoa(binary.Size(bodyBytes)))
		//ctx := zipkin.NewContext(newRequest.Context(), span)
		newRequest = newRequest.WithContext(ctx)

		res, err := client.DoWithAppSpan(newRequest, target)
		if err != nil {
			log.Printf("call to %s returned error: %+v\n", target, err)
			http.Error(w, err.Error(), 500)
			return
		}
		auxBodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyBytes = append(bodyBytes, auxBodyBytes...)
		span.Tag("res-size", strconv.Itoa(binary.Size(auxBodyBytes)))

		res.Body.Close()

	}
	w.Write(bodyBytes)

}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

//func floatNormalDistribution(mean float64, dev float64) float64 {
//  return rand.NormFloat64() * dev + mean
//}

func RandomTarget(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		CallRandomTarget(w, r, client, targets[:], requestSize, calculationTime, load)
	}
}

func CallRandomTarget(w http.ResponseWriter, r *http.Request,
	client *zipkinhttp.Client, targets []string, requestSize uint,
	calculationTime uint, load float64) {

	span, ctx := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
	//span := zipkin.SpanFromContext(r.Context())

	if span == nil {
		log.Printf("span is nil")
	}

	span.Tag("termination_node", "false")

	FinityCpuUsage(calculationTime, load)
	span.Annotate(time.Now(), "expensive_calc_done")

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	target := randomSelection(targets[:])

	log.Printf("-->" + target)
	newRequest, err := http.NewRequest("POST", "http://"+target+":8080/random", bytes.NewBuffer(byteMessage))
	span.Tag("source", globalName)
	span.Tag("target", target)
	span.Tag("req-size", strconv.Itoa(binary.Size(byteMessage)))

	if err != nil {
		log.Printf("unable to create client: %+v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer span.Finish()

	byteMessage = nil

	ctx = zipkin.NewContext(newRequest.Context(), span)
	newRequest = newRequest.WithContext(ctx)

	res, err := client.DoWithAppSpan(newRequest, "random "+target)
	if err != nil {
		log.Printf("call to %s returned error: %+v\n", target, err)
		http.Error(w, err.Error(), 500)
		return
	}
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	span.Tag("res-size", strconv.Itoa(binary.Size(bodyBytes)))
	res.Body.Close()
	w.Write(bodyBytes)
}

func randomSelection(targets []string) string {
	size := len(targets)

	if size <= 0 {
		return "localhost"
	}

	selected := targets[rand.Intn(size)]
	return selected
}

func TerminationRequest(client *zipkinhttp.Client, name string, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Printf("into %s", name)
		log.Printf("%+v", targets)
		//query := r.URL.Query().Get("query")

		if len(targets) <= 0 {

			log.Printf("termination")

			byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))
			w.Header().Set("Content-Type", "octect-stream")
			w.Write(byteMessage)

			span, _ := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
			//span := zipkin.SpanFromContext(r.Context())
			span.Tag("termination_node", "true")
			//span.Tag("source", globalName)
			//span.Tag("target", "terminal")
			//span.Tag("req-size", strconv.Itoa(binary.Size(byteMessage)))
			timeElapsed := FinityCpuUsage(calculationTime, load)
			span.Tag("elapsed_time_ms", strconv.FormatInt(int64(timeElapsed/time.Millisecond), 10))
			// the timeElapsed above must be Time
			// span.Annotate(timeElapsed, "time elapsed")
			span.Finish()

		}
	}
}
