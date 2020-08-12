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

//var globalName = ""
//var globalPort = "8080"
//var bufferReader *Queue
var (
	bufferReader *Queue
	globalName string
	globalPort string
	port	   int
	name       string
	msgSize    uint
	msgTime    uint
	randomSeed int64
	x          int
	y          int
	a          float64
	b          float64
	c          float64
	d          float64
	e          float64
	f          float64
	g          float64
	h          float64
	zipkin_black_hole string
	sampling_black_hole string
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
	load := getCpuUsage(x, y, a, b, c, d, e, f, g, h)
	microservice.RequestsPerSecond = int(load / (float64(microservice.ProcessTime) / 1000.0))
	log.Printf("%+v\n", microservice)

	bufferReader = NewQueue(microservice.RequestsPerSecond)

	SetMemUsage(x, y, a, b, c, d, e, f, g, h)

	r := mux.NewRouter()


	r.Methods("POST").Path("/all").HandlerFunc(callAllTargets("all", microservice, addrs))
	r.Methods("POST").Path("/random").HandlerFunc(callRandomTargets("random", microservice, addrs))

	for key, _ := range generatedRouteMap {
		log.Printf("creating endpoint /%s\n", key)
		r.Methods("POST").Path("/"+key).HandlerFunc(handleRequest(name, key, microservice))
	}

	//If this microservice has dependencies
	//if len(addrs) != 0 {
	//	log.Printf("setting non-terminal handlers in %s", name)
	//
	//	r.Methods("POST").Path("/").HandlerFunc(AllTargets(addrs[:], msgSize, msgTime, microservice, x, y, a, b, c, d, e, f, g, h))
	//	r.Methods("POST").Path("/random").HandlerFunc(RandomTarget(addrs[:], msgSize, msgTime, microservice, x, y, a, b, c, d, e, f, g, h))
	//
	//} else {
	//
	//	log.Printf("setting terminal handlers in %s", name)
	//
	//	r.Methods("POST").Path("/").HandlerFunc(TerminationRequest(name, addrs[:], msgSize, msgTime, microservice, x, y, a, b, c, d, e, f, g, h))
	//	r.Methods("POST").Path("/random").HandlerFunc(TerminationRequest(name, addrs[:], msgSize, msgTime, microservice, x, y, a, b, c, d, e, f, g, h))
	//
	//}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":"+globalPort,
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
	load := FinityCpuUsage(uint(service.ProcessTime), x, y, a, b, c, d, e, f, g, h)
	fakeBody := make([]byte, integerNormalDistribution(msgSize, 10))
	log.Printf("processing... cpu:%+v body_size:%d\n", load, len(fakeBody))
	return fakeBody
}

func integerNormalDistribution(mean uint, dev uint) uint {
	return uint(math.Round(rand.NormFloat64()*float64(dev))) + mean
}

func callNext(target string, requestType string, service *Service, w http.ResponseWriter, r *http.Request) ([]byte, int) {
	req := new(Request)

	//Set 'Value' as the location that sent the request
	req.Value = r.RemoteAddr
	log.Println("-- buffering to queue -- ")
	bufferReader.Push(req)

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
			return []byte{0}, http.StatusBadGateway
			w.Header().Set("ST-Size-Bytes", "0")
		}
	} else {
		w.Header().Set("ST-Termination", "true")
		w.Header().Set("ST-Size-Bytes", strconv.Itoa(len(body)))
	}

	log.Println(" -- unbuffering -- ")
	bufferReader.Pop()
	return body, http.StatusOK
}

func handleRequest(name string, requestType string, service *Service) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		target := getNextTarget(name, requestType)
		w.Header().Set("Content-Type", "application/octet-stream")

		// add go routine to call next
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

func callAllTargets(requestType string, service *Service, addrs []string) http.HandlerFunc  {

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

func callRandomTargets(requestType string, service *Service, addrs []string) http.HandlerFunc  {

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





// Traverse the path defined for that requestType in the generatedRouteMap
func traverseTargets(name string, w http.ResponseWriter, r *http.Request,
	targets []string, requestSize uint,
	calculationTime uint, load float64, x int, y int, requestType string,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) (int, error) {

	cpuRequest := FinityCpuUsage(calculationTime, x, y, a, b, c, d, e, f, g, h)
	log.Printf("cpu load %f\n", cpuRequest)

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))
	log.Printf("message size %d B\n", len(byteMessage))

	target := getNextTarget(name, requestType)
	w.Header().Set("Content-Type", "octect-stream")
	w.Header().Set("Next-Hop", target)

	if target != "" {
		log.Printf("-->" + target)
		newRequest, err := http.NewRequest("POST", "http://"+target+":"+globalPort+"/"+requestType, bytes.NewBuffer(byteMessage))
		if err != nil {
			log.Panic("unable to create client: %+v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		bodyBytes, _ := ioutil.ReadAll(newRequest.Body)
		return w.Write(bodyBytes)
	} else {
		log.Printf("termination node")
		FinityCpuUsage(calculationTime, x, y, a, b, c, d, e, f, g, h)
		return w.Write(byteMessage)
	}
}

func AllTargets(targets []string, requestSize uint, calculationTime uint, serv *Service, x int, y int,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		log.Printf("ok")
		_, error := CallAllTargets(w, r, targets[:], requestSize, calculationTime, serv, x, y, a, b, c, d, e, f, g, h)
		if error != nil {
			http.Error(w, error.Error(), http.StatusInternalServerError)
			log.Panic(error)
		}

	}
}

func CallAllTargets(w http.ResponseWriter, r *http.Request, targets []string, requestSize uint, calculationTime uint, serv *Service, x int, y int,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) (int, error){
	log.Println("-- calling all -- ")

	FinityCpuUsage(calculationTime, x, y, a, b, c, d, e, f, g, h)
	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	log.Println("-- instantiating new requests -- ")
	req := new(Request)

	//Set 'Value' as the location that sent the request
	req.Value = r.RemoteAddr
	log.Println("-- bufering to queue -- ")
	bufferReader.Push(req)
	log.Println("-- sleeping -- ")

	//Simulate the process time for this request after being pushed on the queue by using the sleep method
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)

	log.Println(" -- unbuffering -- ")
	bufferReader.Pop()

	chn := make(chan byte, len(targets))
	for _, target := range targets {
		log.Printf("Sending Request to target: %s", target)

		go func(ch chan byte) {
			body, err := sendRequest(w, r, target, requestSize, calculationTime, byteMessage, byteMessage, a, b, c, d, e, f, g, h)
			chn <- byte(body)
			if err != nil {
				w.Write(make([]byte, 0))
				http.Error(w, err.Error(), http.StatusBadGateway)
				log.Panic(err)
			}
		}(chn)
	}
	log.Println("-- writing body -- ")
	// BE CAREFUL! The bodyBytes aren't filled with the values from the requests. It is missing some sync here to
	// wait data from all requests before write out the response
	bodybytes := byte(0)
	for range targets {
		bodybytes += <- chn
	}
	return w.Write(make([]byte, bodybytes))

}


//func floatNormalDistribution(mean float64, dev float64) float64 {
//  return rand.NormFloat64() * dev + mean
//}

func sendRequest(w http.ResponseWriter, r *http.Request, target string, requestSize uint,
	calculationTime uint, bodyBytes []byte, byteMessage []byte,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) (int, error)  {

	log.Println("** sending request **")
	log.Printf("-->" + target)
	log.Println("** new request **")
	newRequest, err := http.NewRequest("POST", "http://"+target+":"+globalPort+"/", bytes.NewBuffer(byteMessage))

	if err != nil {
		log.Printf("unable to create client: %+v\n", err)
		//http.Error(w, err.Error(), 500)
		return -1, err
	}

	auxBodyBytes, _ := ioutil.ReadAll(newRequest.Body)
	bodyBytes = append(bodyBytes, auxBodyBytes...)

	return w.Write(bodyBytes)
}

func RandomTarget(targets []string, requestSize uint, calculationTime uint, serv *Service, x int, y int,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		CallRandomTarget(w, r, targets[:], requestSize, calculationTime, serv, x, y, a, b, c, d, e, f, g, h)
	}
}

func CallRandomTarget(w http.ResponseWriter, r *http.Request,
	targets []string, requestSize uint,
	calculationTime uint, serv *Service, x int, y int,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) {

	req := new(Request)

	//Set 'Value' as the location that sent the request
	req.Value = r.RemoteAddr

	bufferReader.Push(req)

	//Simulate the process time for this request after being pushed on the queue by using the sleep method
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)

	bufferReader.Pop()

	FinityCpuUsage(calculationTime, x, y, a, b, c, d, e, f, g, h)

	byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))

	target := randomSelection(targets[:])

	log.Printf("-->" + target)
	log.Printf("Sending Request to target: %s", target)

	//Simulate the process time performed by the Microservice
	time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)

	newRequest, err := http.NewRequest("POST", "http://"+target+":"+globalPort+"/random", bytes.NewBuffer(byteMessage))

	if err != nil {
		log.Printf("unable to create client: %+v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}

	byteMessage = nil

	bodyBytes, _ := ioutil.ReadAll(newRequest.Body)
	w.Write(bodyBytes)
}



func TerminationRequest(name string, targets []string, requestSize uint, calculationTime uint, serv *Service, x int, y int,
	a float64, b float64, c float64, d float64, e float64, f float64, g float64, h float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Printf("into %s", name)
		log.Printf("%+v", targets)

		if len(targets) <= 0 {

			log.Printf("termination")
			req := new(Request)

			//Set 'Value' as the location that sent the request
			req.Value = r.RemoteAddr
			load := getCpuUsage(x, y, a, b, c, d, e, f, g, h)

			log.Printf("Currently in: %s", globalName)
			log.Printf("The load: %f", load)

			bufferReader.Push(req)

			//Simulate the process time for this request after being pushed on the queue by using the sleep method
			time.Sleep(time.Duration(serv.ProcessTime) * time.Millisecond)

			bufferReader.Pop()

			byteMessage := make([]byte, integerNormalDistribution(requestSize, 10))
			w.Header().Set("Content-Type", "octect-stream")
			w.Write(byteMessage)

			FinityCpuUsage(calculationTime, x, y, a, b, c, d, e, f, g, h)
		}
	}
}
