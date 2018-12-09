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
	Execute(destination string, payloadSize uint64) []byte
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

type ProcessBeforeRequest struct {
	InnerBehaviorBase
}

func (r ProcessBeforeRequest) Execute(destination string, payloadSize uint64) []byte {
	r.doSomething()
	result := request("", 0)
	return result
}

type RequestBeforeProcess struct {
	InnerBehaviorBase
}

func (r RequestBeforeProcess) Execute(destination string, payloadSize uint64) []byte {
	result := request("", 0)
	r.doSomething()
	return result
}

func request(destination string, payloadSize uint64) []byte {
	if len(destination) <= 0 {
		return make([]byte, payloadSize)
	}

	payload := bytes.NewReader(make([]byte, payloadSize))

	response, err := http.Post(destination, mimeType, payload)
	if err != nil {
		log.Fatalf("could not post: %v", err)
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return bytes

}

type ExternalBehavior struct {
	lastChild int
}

func (base ExternalBehavior) RoundRobin(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	result := innerBehavior.Execute(children[base.lastChild], payload)
	base.lastChild = (len(children) + base.lastChild + 1) % len(children)
	return result
}

func (base ExternalBehavior) Random(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	next := rand.Intn(len(children))
	result := innerBehavior.Execute(children[next], payload)
	return result
}

func (base ExternalBehavior) All(children []string, payload uint64, innerBehavior InnerBehavior) []byte {
	resultSize := 0
	for i := 0; i < len(children); i++ {
		result := innerBehavior.Execute(children[i], payload)
		resultSize += len(result)
	}

	return make([]byte, resultSize/len(children))
}
