FROM golang

RUN go get github.com/google/uuid && \
	go get github.com/gorilla/mux && \
    go get github.com/shirou/gopsutil/process

ADD . /src/microservice

RUN cd /src/microservice && make microservice


FROM bitnami/minideb:latest
WORKDIR /home
COPY --from=0 /src/microservice/microservice /usr/local/bin/
EXPOSE 8000-8999
ENTRYPOINT ["/usr/local/bin/microservice"]
