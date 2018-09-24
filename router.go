package main

import (
	"bytes"
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

func main() {
	writePid()
	var (
		name       string
		msgSize    uint
		load       float64
		msgTime    uint
		memUsage   uint
		zipkinAddr string
	)

	flag.StringVar(&zipkinAddr, "zipkin", "0.0.0.0:9411", "zipkin addr:port default 0.0.0.0:9411")
	flag.StringVar(&name, "name", "", "service name")
	flag.UintVar(&msgSize, "msg-size", 256, "average size in bytes default:256")
	flag.Float64Var(&load, "load", 0.1, "CPU load per message default:10% (0.1)")
	flag.UintVar(&msgTime, "msg-time", 10, "Time do compute an msg-request default 10ms")
	flag.UintVar(&memUsage, "mem", 128, "min memory usage default:128MB")
	flag.Parse()
	addrs := flag.Args()

	if len(name) <= 0 {
		log.Fatal("argument --name must be set")
	}

	tracer, err := newTracer(name, zipkinAddr)
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

	r.Methods("POST").Path("/all").HandlerFunc(AllTargets(client, addrs[:], msgSize, msgTime, load))
	r.Methods("POST").Path("/random").HandlerFunc(RandomTarget(client, addrs[:], msgSize, msgTime, load))
	r.Methods("POST").Path("/").HandlerFunc(TerminationRequest(client, name, addrs[:], msgSize, msgTime, load))

	r.Use(zipkinhttp.NewServerMiddleware(
		tracer,
		zipkinhttp.SpanName("request")), // name for request span
	)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func writePid() {
	pid := os.Getpid()
	bpid := []byte(strconv.Itoa(pid))
	ioutil.WriteFile("/tmp/go.pid", bpid, 0644)
}

func AllTargets(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		CallAllTargets(w, r, client, targets[:], requestSize, calculationTime)
	}
}

func CallAllTargets(w http.ResponseWriter, r *http.Request, client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint) {
	span := zipkin.SpanFromContext(r.Context())
	span.Tag("terimation_node", "false")

	time.Sleep(time.Duration(integerNormalDistribution(calculationTime, 10)) * time.Millisecond)
	span.Annotate(time.Now(), "foo_expensive_calc_done")

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	for _, target := range targets {

		newRequest, err := http.NewRequest("POST", "http://"+target+":8080?query=all", bytes.NewBuffer(byteMessage))
		if err != nil {
			log.Printf("unable to create client: %+v\n", err)
			// http.Error(w, err.Error(), 500)
			// return
		}

		byteMessage = nil

		ctx := zipkin.NewContext(newRequest.Context(), span)
		newRequest = newRequest.WithContext(ctx)

		res, err := client.DoWithAppSpan(newRequest, target+"_target")
		if err != nil {
			log.Printf("call to %s returned error: %+v\n", target, err)
			// http.Error(w, err.Error(), 500)
			// return
		}
		res.Body.Close()

	}

}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

//func floatNormalDistribution(mean float64, dev float64) float64 {
//  return rand.NormFloat64() * dev + mean
//}

func RandomTarget(client *zipkinhttp.Client, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		CallRandomTarget(w, r, client, targets[:], requestSize, calculationTime)
	}
}

func CallRandomTarget(w http.ResponseWriter, r *http.Request,
	client *zipkinhttp.Client, targets []string, requestSize uint,
	calculationTime uint) {

	span := zipkin.SpanFromContext(r.Context())
	span.Tag("terimation_node", "false")

	time.Sleep(time.Duration(integerNormalDistribution(calculationTime, 10)) * time.Millisecond)
	span.Annotate(time.Now(), "foo_expensive_calc_done")

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	target := randomSelection(targets[:])

	newRequest, err := http.NewRequest("POST", "http://"+target+":8080?query=random", bytes.NewBuffer(byteMessage))
	if err != nil {
		log.Printf("unable to create client: %+v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}

	byteMessage = nil

	ctx := zipkin.NewContext(newRequest.Context(), span)
	newRequest = newRequest.WithContext(ctx)

	res, err := client.DoWithAppSpan(newRequest, target+"_target")
	if err != nil {
		log.Printf("call to %s returned error: %+v\n", target, err)
		// http.Error(w, err.Error(), 500)
		// return
	}
	res.Body.Close()

}

func randomSelection(targets []string) string {
	size := len(targets)

	if size <= 0 {
		return "none"
	}

	selected := targets[rand.Intn(size)]
	return selected
}

func TerminationRequest(client *zipkinhttp.Client, name string, targets []string, requestSize uint, calculationTime uint, load float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("into %s", name)
		log.Printf("%+v", targets)
		query := r.URL.Query().Get("query")
		if len(targets) <= 0 {
			log.Printf("termination")

			span := zipkin.SpanFromContext(r.Context())
			span.Tag("termination_node", "true")
			span.Tag("request_size", strconv.FormatInt(r.ContentLength, 10))

			timeElapsed := FinityCpuUsage(calculationTime, load)
			span.Tag("elapsed_time_ms", strconv.FormatInt(int64(timeElapsed/time.Millisecond), 10))
			// the timeElapsed above must be Time
			// span.Annotate(timeElapsed, "time elapsed")

		} else {
			log.Printf("next call : %s", query)

			if query == "random" {
				CallRandomTarget(w, r, client, targets[:], requestSize, calculationTime)
			} else {
				CallAllTargets(w, r, client, targets[:], requestSize, calculationTime)
			}

		}
	}
}
