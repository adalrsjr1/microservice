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

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////NEW STRUCTURE////////////////////////////////////////////////////////////////////////////////////////////
/**
* Create a Service structure that will have the following attributes:
* ID- the name of this particular Service
* RequestsPerSecond- An integer value that represents the number of requests this microsecond can make per second
* ProcessTime- An integer value that represents the amount of time this Microservice spends on Processing a request
**/
type Service struct {
	ID                string
	RequestsPerSecond int
	ProcessTime       int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var globalName = ""
var tracer *zipkin.Tracer
var bufferReader *Queue

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
		x          float64
		y          float64
	)
	flag.StringVar(&zipkinAddr, "zipkin", "0.0.0.0:9411", "zipkin addr:port default 0.0.0.0:9411")
	flag.StringVar(&name, "name", "", "service name")
	flag.UintVar(&msgSize, "msg-size", 256, "average size in bytes default:256")
	flag.Float64Var(&load, "load", 0.1, "CPU load per message default:10% (0.1)")
	flag.UintVar(&msgTime, "msg-time", 10, "Time do compute an msg-request default 10ms")
	flag.UintVar(&memUsage, "mem", 128, "min memory usage default:128MB")
	flag.BoolVar(&isSampling, "sampling", true, "sampling messages to store into zipkin")
	flag.Float64Var(&x, "x", 0, "x value")
	flag.Float64Var(&y, "y", 0, "y value")
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

	//Create an object that reflects the service executing this code
	//Should have a name, microservices that it depends on, the number of dependencies, and the number of requests that it can handle
	microservice := new(Service)
	microservice.ID = name
	microservice.ProcessTime = int(msgTime)

	//calculate the number of requests per second that are handled on average based on the CPU load and processing time
	microservice.RequestsPerSecond = int((load * 100.0) / float64((float64(microservice.ProcessTime) / 1000.0)))

	bufferReader = NewQueue(microservice.RequestsPerSecond)

	SetMemUsage(x, y)

	r := mux.NewRouter()

	//If this microservice has dependencies
	if len(addrs) != 0 {
		log.Printf("setting non-terminal handlers in %s", name)

		r.Methods("POST").Path("/").HandlerFunc(AllTargets(client, addrs[:], msgSize, msgTime, load, microservice, x, y))
		r.Methods("POST").Path("/random").HandlerFunc(RandomTarget(client, addrs[:], msgSize, msgTime, load, microservice, x, y))

		r.Methods("POST").Path("/0").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "0"))
		r.Methods("POST").Path("/1").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "1"))
		r.Methods("POST").Path("/2").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "2"))
		r.Methods("POST").Path("/3").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "3"))
		//If this microservice has no dependencies
	} else {

		log.Printf("setting terminal handlers in %s", name)

		r.Methods("POST").Path("/").HandlerFunc(TerminationRequest(client, name, addrs[:], msgSize, msgTime, load, microservice, x, y))
		r.Methods("POST").Path("/random").HandlerFunc(TerminationRequest(client, name, addrs[:], msgSize, msgTime, load, microservice, x, y))

		r.Methods("POST").Path("/0").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "0"))
		r.Methods("POST").Path("/1").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "1"))
		r.Methods("POST").Path("/2").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "2"))
		r.Methods("POST").Path("/3").HandlerFunc(handleRequest(name, client, addrs[:], msgSize, msgTime, load, x, y, "3"))
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

	nextNode := generatedRouteMap[requestType][currentNode]

	return nextNode
}

func handleRequest(name string, client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64, x float64, y float64, requestType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ok")
		traverseTargets(name, w, r, client, targets[:], requestSize, calculationTime, load, x, y, requestType)
	}
}

// Traverse the path defined for that requestType in the generatedRouteMap
func traverseTargets(name string, w http.ResponseWriter, r *http.Request,
	client *zipkinhttp.Client, targets []string, requestSize uint,
	calculationTime uint, load float64, x float64, y float64, requestType string) {

	span, ctx := tracer.StartSpanFromContext(r.Context(), "size")

	if span == nil {
		log.Printf("span is nil")
	}

	span.Tag("termination_node", "false")

	FinityCpuUsage(calculationTime, x, y)
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
		timeElapsed := FinityCpuUsage(calculationTime, x, y)
		span.Tag("elapsed_time_ms", strconv.FormatInt(int64(timeElapsed/time.Millisecond), 10))

		span.Finish()
	}
}

func AllTargets(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64, serv *Service, x float64, y float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ok")
		CallAllTargets(w, r, client, targets[:], requestSize, calculationTime, load, serv, x, y)
	}
}

func CallAllTargets(w http.ResponseWriter, r *http.Request, client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64, serv *Service, x float64, y float64) {

	FinityCpuUsage(calculationTime, x, y)
	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	var bodyBytes []byte

	req := new(Request)

	//Set 'Value' as the location that sent the request
	req.Value = r.RemoteAddr

	bufferReader.Push(req)

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////NEW ADDITION TO THIS FUNCTION/////////////////////////////////////////////////////////////////////////////
	//Simulate the process time for this request after being pushed on the queue by using the sleep method
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	bufferReader.Pop()

	for _, target := range targets {
		log.Printf("Sending Request to target: %s", target)
		go sendRequest(w, r, client, target, requestSize, calculationTime, load, bodyBytes, byteMessage)

	}

	// BE CAREFUL! The bodyBytes aren't filled with the values from the requests. It is missing some sync here to
	// wait data from all requests before write out the response
	w.Write(bodyBytes)

}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

func sendRequest(w http.ResponseWriter, r *http.Request, client *zipkinhttp.Client, target string, requestSize uint, calculationTime uint, load float64, bodyBytes []byte, byteMessage []byte) {

	//span := tracer.StartSpan("size")
	span, _ := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
	span.Tag("termination_node", "false")
	log.Printf("-->" + target)

	newRequest, err := http.NewRequest("POST", "http://"+target+":8080/", bytes.NewBuffer(byteMessage))
	span.Tag("source", globalName)
	span.Tag("target", target)
	span.Tag("req-size", strconv.Itoa(binary.Size(bodyBytes)))

	if err != nil {
		log.Printf("unable to create client: %+v\n", err)
		http.Error(w, err.Error(), 500)
	}
	defer span.Finish()
	span.Tag("req-size", strconv.Itoa(binary.Size(bodyBytes)))
	//ctx := zipkin.NewContext(newRequest.Context(), span)
	//newRequest = newRequest.WithContext(ctx)

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

//func floatNormalDistribution(mean float64, dev float64) float64 {
//  return rand.NormFloat64() * dev + mean
//}

func RandomTarget(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64, serv *Service, x float64, y float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		CallRandomTarget(w, r, client, targets[:], requestSize, calculationTime, load, serv, x, y)
	}
}

func CallRandomTarget(w http.ResponseWriter, r *http.Request,
	client *zipkinhttp.Client, targets []string, requestSize uint,
	calculationTime uint, load float64, serv *Service, x float64, y float64) {

	req := new(Request)

	//Set 'Value' as the location that sent the request
	req.Value = r.RemoteAddr

	bufferReader.Push(req)

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////NEW ADDITION TO THIS FUNCTION/////////////////////////////////////////////////////////////////////////////
	//Simulate the process time for this request after being pushed on the queue by using the sleep method
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	bufferReader.Pop()

	span, ctx := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
	//span := zipkin.SpanFromContext(r.Context())

	if span == nil {
		log.Printf("span is nil")
	}

	span.Tag("termination_node", "false")

	FinityCpuUsage(calculationTime, x, y)
	span.Annotate(time.Now(), "expensive_calc_done")

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	target := randomSelection(targets[:])

	log.Printf("-->" + target)
	log.Printf("Sending Request to target: %s", target)

	//Simulate the process time performed by the Microservice
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)

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

func TerminationRequest(client *zipkinhttp.Client, name string, targets []string, requestSize uint, calculationTime uint, load float64, serv *Service, x float64, y float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Printf("into %s", name)
		log.Printf("%+v", targets)
		//query := r.URL.Query().Get("query")

		if len(targets) <= 0 {

			log.Printf("termination")
			req := new(Request)

			//Set 'Value' as the location that sent the request
			req.Value = r.RemoteAddr

			log.Printf("Currently in: %s", globalName)
			log.Printf("The load: %f", load)

			bufferReader.Push(req)

			///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
			//////////////////////////////////////////////////////////NEW ADDITION TO THIS FUNCTION/////////////////////////////////////////////////////////////////////////////
			//Simulate the process time for this request after being pushed on the queue by using the sleep method
			time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)
			///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
			///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

			bufferReader.Pop()

			byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))
			w.Header().Set("Content-Type", "octect-stream")
			w.Write(byteMessage)

			span, _ := tracer.StartSpanFromContext(r.Context(), "size") //zipkin.SpanFromContext(r.Context())
			//span := zipkin.SpanFromContext(r.Context())
			span.Tag("termination_node", "true")
			//span.Tag("source", globalName)
			//span.Tag("target", "terminal")
			//span.Tag("req-size", strconv.Itoa(binary.Size(byteMessage)))

			timeElapsed := FinityCpuUsage(calculationTime, x, y)
			span.Tag("elapsed_time_ms", strconv.FormatInt(int64(timeElapsed/time.Millisecond), 10))
			// the timeElapsed above must be Time
			// span.Annotate(timeElapsed, "time elapsed")
			span.Finish()

		}
	}
}
