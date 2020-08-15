package main

import (
	"bytes"
	"flag"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
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
	globalPort = strconv.Itoa(port)

	log.Printf("listening on %s", name)

	rand.Seed(randomSeed)

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
	fakeBody := make([]byte, integerNormalDistribution(msgSize, 10))
	log.Printf("processing... body_size:%d, service:%+v\n", len(fakeBody), service)
	log.Printf(" --- ## %+v ## --- \n", service)
	return fakeBody
}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

func callNext(target string, requestType string, service *Service, w http.ResponseWriter, r *http.Request) ([]byte, int) {
	<-throttling
	log.Println("-- buffering to queue -- ")

	body := doSomething(service)
	if target != "" {
		w.Header().Set("Next-Hop", target)
		w.Header().Set("ST-Termination", "false")
		log.Printf("Callling next %s\n", "http://"+target+":"+globalPort+"/"+requestType)
		resp, err := http.Post("http://"+target+":"+globalPort+"/"+requestType, "application/octet-stream", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("Error calling %s\n", "http://"+target+":"+globalPort+"/"+requestType)
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

	return func(w http.ResponseWriter, r *http.Request) {
		target := getNextTarget(name, requestType)
		w.Header().Set("Content-Type", "application/octet-stream")

		body, httpStatus := callNext(target, requestType, service, w, r)
		w.WriteHeader(httpStatus)
		w.Write(body)
		log.Printf("handling request %s\n", requestType)

	}

}

func getNextTarget(currentNode string, requestType string) string {
	nextNode := generatedRouteMap[requestType][currentNode]
	return nextNode
}

func callAllTargets(requestType string, service *Service, addrs []string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
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
			auxBody, auxHttpStatus := callNext(target, requestType, service, w, r)
			if auxHttpStatus != http.StatusOK {
				log.Printf("HTTP ERROR %d when calling %s\n", auxHttpStatus, target)
				w.WriteHeader(auxHttpStatus)
				w.Write([]byte{0})
				return
			}

			body = append(body, auxBody...)
		}

		w.WriteHeader(httpStatus)
		w.Write(body)
		log.Printf("handling request to all children")
	}

}

func callRandomTargets(requestType string, service *Service, addrs []string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		httpStatus := http.StatusOK
		body := []byte{}

		target := randomSelection(addrs)

		// add go routine to call next
		auxBody, auxHttpStatus := callNext(target, requestType, service, w, r)
		if auxHttpStatus != http.StatusOK {
			log.Printf("HTTP ERROR %d when calling %s\n", auxHttpStatus, target)
			w.WriteHeader(httpStatus)
			w.Write(body)
			return
		}

		body = append(body, auxBody...)
		w.WriteHeader(httpStatus)
		w.Write(body)

		log.Printf("handling request to random %s", target)
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
