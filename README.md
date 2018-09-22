# Dummy microservice

A tiny golang application simulating a microservice. It

- listens for inbound connections
- connects to other named microservices specified on the command line
- sends messages with size specified on the command line
- maintains a user defined CPU and memory load
- logs sent and received through Zipkin
- is a REST microservice

## Build

### Using Go natively

```bash
make microservice
```

## Usage

```bash
Usage of ./micro-sock:
  -name string
        service name
  -zipkin string
        zipkin address (addrs:port) -- default 0.0.0.0:9411
  -msg-size uint
        average size of all messages outgoing -- default:256
  -load float64
        average CPU load per request being processing -- default 10% of CPU
  -msg-time uint
        average time to process an incoming message -- default 10ms
  -mem uint
        minimum memory used by the microservice -- default 128MB
```

## Example

The following example will start microservice and attempt to connect to a different instance of micro-sock, on host "test-host". In this case, zipkin is on host "zipkin"

```bash
./microservice -name=micro -zipkin=zipkin:9411 test-host
```
