FROM alpine
WORKDIR /home
ADD ./microservice /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/microservice"]
