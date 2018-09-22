.PHONY: all image clean publish

IMAGE=adalrsjr1/microservice

all: image

microservice: router.go tracer.go metrics.go
	env GOOS=linux GOARCH=amd64 go build -tags netgo

image: Dockerfile microservice
	docker build -t $(IMAGE) .

clean:
	rm -f microservice
	docker rmi -f $(IMAGE) 2>/dev/null || true

publish:
	docker push $(IMAGE)
