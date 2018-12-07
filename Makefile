.PHONY: all image clean publish

IMAGE=adalrsjr1/microservice

all: clean image publish

microservice: core.go
	env GOOS=linux GOARCH=amd64 CGO_ENABLED="1" go build -tags netgo .

windows: core.go
	env GOOS=linux GOARCH=amd64 CGo_ENABLED="1" go build -tags netgo .

image: Dockerfile microservice
	docker build -t $(IMAGE) .

clean:
	rm -f microservice
	docker rmi -f $(IMAGE) 2>/dev/null || true

publish:
	docker push $(IMAGE)
