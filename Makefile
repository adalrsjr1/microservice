.PHONY: all image clean publish

IMAGE=adalrsjr1/microservice

all: clean microservice image publish

microservice: router.go metrics.go  queue.go routeMap.go tracer.go
	env GOOS=linux GOARCH=amd64 go build -tags netgo

image: Dockerfile microservice
	docker build -t $(IMAGE) .

clean:
	rm -f microservice
	docker rmi -f $(IMAGE) 2>/dev/null || true

publish:
	docker push $(IMAGE)
