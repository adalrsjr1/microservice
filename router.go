package main

import (
	"bytes"
	"flag"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-lib/metrics"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

/**
* Create a Service structure that will have the following attributes:
* ID- the name of this particular Service
* RequestsPerSecond- An integer value that represents the number of requests this microsecond can make per second
* ProcessTime- An integer value that represents the amount of time this Microservice spends on Processing a request
**/
type Service struct {
	ID                string
	RequestsPerSecond float64
	ProcessTime       int
}

var (
	globalName          string
	globalPort          string
	port                int
	name                string
	msgSize             uint
	msgTime             uint
	randomSeed          int64
	x                   int
	y                   int
	a                   float64
	b                   float64
	c                   float64
	d                   float64
	e                   float64
	f                   float64
	g                   float64
	h                   float64
	zipkin_black_hole   string
	sampling_black_hole string
	throttling          <-chan time.Time
	root				bool
)

func main() {
	writePid()

	flag.StringVar(&zipkin_black_hole, "zipkin", "", "blackhole")
	flag.StringVar(&sampling_black_hole, "sampling", "", "blackhole")
	flag.StringVar(&name, "name", "", "service name")
	flag.IntVar(&port, "port", 8080, "port")
	flag.Int64Var(&randomSeed, "random-seed", 42, "random seed")
	flag.UintVar(&msgSize, "msg-size", 256, "average size in bytes default:256")
	flag.UintVar(&msgTime, "msg-time", 10, "Time do compute an msg-request default 10ms")
	flag.IntVar(&x, "x", 0, "parameter X")
	flag.IntVar(&y, "y", 0, "parameter Y")
	flag.Float64Var(&a, "a", 0, "parameter A")
	flag.Float64Var(&b, "b", 0, "parameter B")
	flag.Float64Var(&c, "c", 0, "parameter C")
	flag.Float64Var(&d, "d", 0, "parameter D")
	flag.Float64Var(&e, "e", 0, "parameter E")
	flag.Float64Var(&f, "f", 0, "parameter F")
	flag.Float64Var(&g, "g", 0, "parameter G")
	flag.Float64Var(&h, "h", 0, "parameter H")
	flag.Parse()

	addrs := flag.Args()

	if len(name) <= 0 {
		log.Fatal("argument --name must be set")
	}
	globalName = name
	root = false
	if globalName == "svc-0-mock" {
		root = true
	}
	globalPort = strconv.Itoa(port)
	rand.Seed(randomSeed)

	log.Println("setting tracer")
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	sampling, _ := strconv.ParseFloat(sampling_black_hole, 64)
	cfg := jaegercfg.Configuration{
		ServiceName: name,
		Sampler:     &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: sampling,
		},
		//LocalAgentHostPort instructs reporter to send spans to jaeger-agent at this address. Can be provided by FromEnv() via the environment variable named JAEGER_AGENT_HOST / JAEGER_AGENT_PORT
		Reporter:    &jaegercfg.ReporterConfig{
			LogSpans: true,
			LocalAgentHostPort: zipkin_black_hole,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),

	)
	if err != nil {
		log.Printf("error to initialize tracer: %+v\n", err)
	}
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	log.Printf("listening on %s", name)



	//Create an object that reflects the service executing this code
	//Should have a name, microservices that it depends on, the number of dependencies, and the number of requests that it can handle
	microservice := new(Service)
	microservice.ID = name
	microservice.ProcessTime = int(msgTime)

	//calculate the number of requests per second that are handled on average based on the CPU load and processing time
	load := getCpuUsage(x, y, a, b, c, d, e, f, g, h) / 100
	// 500 -> 10K
	microservice.RequestsPerSecond = load * 4000
	log.Printf("load: %f, proc time: %d, reqps: %f", load, microservice.ProcessTime, microservice.RequestsPerSecond)

	throttling = time.Tick(1000000000 / time.Duration(microservice.RequestsPerSecond) * time.Nanosecond)

	SetMemUsage(x, y, a, b, c, d, e, f, g, h)

	r := mux.NewRouter()

	r.Methods("POST").Path("/all").HandlerFunc(callAllTargets("all", microservice, addrs))
	r.Methods("GET").Path("/all").HandlerFunc(callAllTargets("all", microservice, addrs))
	r.Methods("POST").Path("/random").HandlerFunc(callRandomTargets("random", microservice, addrs))

	for key, _ := range generatedRouteMap {
		log.Printf("creating endpoint /%s\n", key)
		r.Methods("POST").Path("/" + key).HandlerFunc(handleRequest(name, key, microservice))
	}

	srv := &http.Server{
		Handler: r,
		Addr:    ":" + globalPort,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func writePid() {
	pid := os.Getpid()
	bpid := []byte(strconv.Itoa(pid))
	ioutil.WriteFile("/tmp/"+globalName+"-ms.pid", bpid, 0644)
}

func doSomething(service *Service) []byte {
	log.Println("mocking processing")
	FinityCpuUsage(uint(service.ProcessTime),x,y,a,b,c,d,e,f,g,h)
	fakeBody := make([]byte, integerNormalDistribution(msgSize, 10))
	log.Printf("processing... body_size:%d, service:%+v\n", len(fakeBody), service)
	log.Printf(" --- ## %+v ## --- \n", service)
	return fakeBody
}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

func callNext(target string, requestType string, service *Service, w http.ResponseWriter, tracer *opentracing.Tracer, clientSpan *opentracing.Span) ([]byte, int) {
	log.Println("-- buffering to queue -- ")

	body := doSomething(service)
	if target != "" {
		w.Header().Set("Next-Hop", target)
		w.Header().Set("ST-Termination", "false")

		url := "http://"+target+":"+globalPort+"/"+requestType
		log.Printf("Callling next %s\n", "http://"+target+":"+globalPort+"/"+requestType)
		//resp, err := http.Post(url, "application/octet-stream", bytes.NewBuffer(body))
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

		// Set some tags on the clientSpan to annotate that it's the client span. The additional HTTP tags are useful for debugging purposes.
		ext.SpanKindRPCClient.Set(*clientSpan)
		ext.HTTPUrl.Set(*clientSpan, url)
		ext.HTTPMethod.Set(*clientSpan, "POST")

		// Inject the client span context into the headers
		(*tracer).Inject((*clientSpan).Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		resp, err := http.DefaultClient.Do(req)

		(*tracer).Inject((*clientSpan).Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		if err != nil {
			log.Printf("Error calling %s\n", url)
			w.Header().Set("ST-Size-Bytes", "0")
			return []byte{0}, http.StatusBadGateway
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			body = append(body, responseBody...)
			w.Header().Set("ST-Size-Bytes", strconv.Itoa(len(body)))
		} else {
			log.Printf("HTTP Error %d calling %s\n", resp.StatusCode, "http://"+target+":"+globalPort+"/"+requestType)
			w.Header().Set("ST-Size-Bytes", "0")
			return []byte{0}, http.StatusBadGateway
		}
	} else {
		w.Header().Set("ST-Termination", "true")
		w.Header().Set("ST-Size-Bytes", strconv.Itoa(len(body)))
	}

	log.Println(" -- unbuffering -- ")
	return body, http.StatusOK
}

func handleRequest(name string, requestType string, service *Service) http.HandlerFunc {
	<-throttling
	return func(w http.ResponseWriter, r *http.Request) {
		span, tracer := startSpan(name, requestType, &r.Header)

		target := getNextTarget(name, requestType)
		w.Header().Set("Content-Type", "application/octet-stream")

		body, httpStatus := callNext(target, requestType, service, w, &tracer, &span)
		w.WriteHeader(httpStatus)
		w.Write(body)
		log.Printf("handling request %s\n", requestType)
		defer span.Finish()
	}

}

func startSpan(name string, requestType string, header *http.Header) (opentracing.Span, opentracing.Tracer) {
	var span opentracing.Span
	tracer := opentracing.GlobalTracer()
	if !root {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(*header))
		span = tracer.StartSpan(requestType, ext.RPCServerOption(spanCtx))
	} else {
		span = tracer.StartSpan(requestType)
	}

	return span, tracer
}

func getNextTarget(currentNode string, requestType string) string {
	nextNode := generatedRouteMap[requestType][currentNode]
	return nextNode
}

func callAllTargets(requestType string, service *Service, addrs []string) http.HandlerFunc {
	<-throttling
	return func(w http.ResponseWriter, r *http.Request) {
		span, tracer := startSpan(name, requestType, &r.Header)

		w.Header().Set("Content-Type", "application/octet-stream")
		httpStatus := http.StatusOK
		body := []byte{0}

		if len(addrs) == 0 {
			addrs = []string{""}
		}

		count := len(addrs)
		for _, target := range addrs {
			count--
			log.Printf("calling --> %s", target)
			// add go routine to call next
			auxBody, auxHttpStatus := callNext(target, requestType, service, w, &tracer, &span)
			if auxHttpStatus != http.StatusOK {
				log.Printf("HTTP ERROR %d when calling %s\n", auxHttpStatus, target)
				w.WriteHeader(auxHttpStatus)
				w.Write([]byte{0})
				defer span.Finish()
				return
			}

			body = append(body, auxBody...)
		}

		w.WriteHeader(httpStatus)
		w.Write(body)
		log.Printf("handling request to all children")
		defer span.Finish()
	}

}

func callRandomTargets(requestType string, service *Service, addrs []string) http.HandlerFunc {
	<-throttling
	return func(w http.ResponseWriter, r *http.Request) {
		span, tracer := startSpan(name, requestType, &r.Header)

		w.Header().Set("Content-Type", "application/octet-stream")
		httpStatus := http.StatusOK
		body := []byte{}

		target := randomSelection(addrs)

		// add go routine to call next
		auxBody, auxHttpStatus := callNext(target, requestType, service, w, &tracer, &span)
		if auxHttpStatus != http.StatusOK {
			log.Printf("HTTP ERROR %d when calling %s\n", auxHttpStatus, target)
			w.WriteHeader(httpStatus)
			w.Write(body)
			defer span.Finish()
			return
		}

		body = append(body, auxBody...)
		w.WriteHeader(httpStatus)
		w.Write(body)

		log.Printf("handling request to random %s", target)
		defer span.Finish()
	}

}

func randomSelection(targets []string) string {
	size := len(targets)

	if size <= 0 {
		return ""
	}

	selected := targets[rand.Intn(size)]
	return selected
}
