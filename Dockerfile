FROM alpine
WORKDIR /home
ADD ./microservice /usr/local/bin/
EXPOSE 8888
ENTRYPOINT ["/usr/local/bin/microservice"]
