package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"runtime"

	"github.com/google/uuid"
)

type InnerBehavior interface {
	Execute(destination string, payloadSize uint64, action string) []byte
}

type InnerBehaviorBase struct {
	cpuLoad                     float64
	memoryRequest               uint64
	timeToElaspseInMilliseconds uint
}

func (base InnerBehaviorBase) doSomething() {
	id := uuid.New().String()
	log.Printf("processing %v cpu=%v memory=%v", id, base.cpuLoad, base.memoryRequest)

	nCpus := runtime.NumCPU()

	mem := SetMemUsage(base.memoryRequest)
	load := base.cpuLoad

	timeToElaspseInMilliseconds := base.timeToElaspseInMilliseconds

	for index := 0; index < nCpus-1; index++ {
		go ForceCpuUsage(timeToElaspseInMilliseconds, load)
	}
	ForceCpuUsage(timeToElaspseInMilliseconds, load)

	FreeMemUsed(mem)
	log.Printf("done with %v in %vms", id, timeToElaspseInMilliseconds)
}

func (base InnerBehaviorBase) request(destination string, payloadSize uint64, action string) []byte {
	if len(destination) <= 0 {
		log.Println("--> terminal")
	} else {
		log.Println("--> " + destination)
	}

	if len(destination) <= 0 {
		return make([]byte, payloadSize)
	}

	payload := bytes.NewReader(make([]byte, payloadSize))

	response, err := http.Post("http://"+destination+":8888"+action, mimeType, payload)

	if err != nil {
		log.Fatalf("could not post: %v", err)
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return bytes

}

type ProcessBeforeRequest struct {
	InnerBehaviorBase
}

func (r ProcessBeforeRequest) Execute(destination string, payloadSize uint64, action string) []byte {
	r.doSomething()
	result := r.request(destination, payloadSize, action+"/processAndCall")
	return result
}

type RequestBeforeProcess struct {
	InnerBehaviorBase
}

func (r RequestBeforeProcess) Execute(destination string, payloadSize uint64, action string) []byte {
	result := r.request(destination, payloadSize, action+"/callAndProcess")
	r.doSomething()
	return result
}

type ExternalBehavior struct {
	lastChild int
}

func NewExternalBehavior() *ExternalBehavior {
	externalBehavior := new(ExternalBehavior)
	externalBehavior.lastChild = 0

	return externalBehavior
}

func (base *ExternalBehavior) RoundRobin(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	log.Println(children)

	if len(children) <= 0 {
		return innerBehavior.Execute(children[base.lastChild], payload, "/roundrobin")
	}

	result := innerBehavior.Execute(children[base.lastChild], payload, "/roundrobin")
	base.lastChild = (base.lastChild + 1) % len(children)
	return result
}

func (base *ExternalBehavior) Random(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	log.Println(children)

	if len(children) <= 0 {
		return innerBehavior.Execute("", payload, "/random")
	}

	next := rand.Intn(len(children))
	result := innerBehavior.Execute(children[next], payload, "/random")
	return result
}

func (base *ExternalBehavior) All(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	log.Println(children)
	size := len(children)
	resultSize := 0

	if size <= 0 {
		result := innerBehavior.Execute("", payload, "/all")
		return result
	}

	for i := 0; i < len(children); i++ {
		result := innerBehavior.Execute(children[i], payload, "/all")
		resultSize += len(result)
	}

	return make([]byte, resultSize/size)
}
