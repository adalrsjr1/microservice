see also my [perf](https://github.com/adalrsjr1/perf/) project
# Dummy microservice

A tiny golang application simulating a microservice. It

- listens for HTTP POST inbound connections
- connects to other named microservices specified on the command line
- sends messages with size specified on the command line
- maintains a user defined CPU and memory load
- logs sent and received through Zipkin

## Build

### Using Go natively

```bash
go  get -u
make microservice
```

## Usage

```bash
Usage of ./microservice:
  --name string
        service name
  --zipkin string
        zipkin address (addrs:port) -- default 0.0.0.0:9411
  --msg-size uint
        average size of all messages outgoing -- default:256
  --msg-time uint
        average time to process an incoming message -- default 10ms
  --{a-h} float64
        parameter {A-H} that affects CPU and memory usage -- default 0
  --x int
        parameter X that affects CPU and memory usage -- default 0
  --y int
        parameter Y that affects CPU and memory usage -- default 0
```

## Example

The following example will start a microservice and attempt to connect to a different instance of micro-sock "test-host". In this case, zipkin is on host "zipkin"

```bash
./microservice -name=micro -zipkin=zipkin:9411 test-host
```

#### To test using docker-compose

```bash
docker-compose -f zipkin_docker-compose.yaml -f docker-compose.custom.yml up
```

#### In another bash or on browser

To exercise all endpoints:

```
curl -I -XPOST localhost:8080/
```
or to exercise few endpoints at random
```
curl -I -XPOST localhost:8080/random
```


#### Exercising URL-based predefined paths

After running `uApp-generator.py`, the file `routeMap.go` will be created. This file defines all the uniques paths through the given microservice graph from the start node to a terminal node. 
In order to exercise one of the paths, use any of the 4 commands below.

```
curl -I -XPOST localhost:8080/0
curl -I -XPOST localhost:8080/1
curl -I -XPOST localhost:8080/2
curl -I -XPOST localhost:8080/3
```

If there are more than 4 paths through the graph that you wish to exercise, you can easily add more endpoints (`/4`, `/5`, ...) in `main()` of `router.go` and everything should work accordingly.
No other changes are nessecary to support more paths.

If there are less than 4 paths through the graph, only use the endpoints less than the number of paths. 


## Deep Dive
#### Valid parameter values
All parameters are continuous unless otherwise specified.
```
-4 <= a <= 4
-250 <= b <= 250
-10 <= c <= 10
1E-5 <= d <= 1E-1 (continuous), 10 <= d <= 1E5 | Step: 10, not continuous
-2.5 <= e <= 2.5
5 <= f <= 10, -10 <= f <= -5
-3 <= g <= 3
-25 <= h < 25
-3 <= x, y <= 3 | Step: 1, not continuous
```
All parameters are independent of each other.

#### Functions used to determine CPU and load
![Functions](https://quicklatex.com/cache3/76/ql_be0aa52379850f1f5b576bc689a00e76_l3.png)

#### Raw functions
If the image ever breaks, here are the raw functions (as LaTeX):
```
Beale: \textrm{CPU} = \frac{(1.5-x+xy)^2 + (2.25-x+xy^2)^2 + (2.625-x+xy^3)^2}{890000} + 0.2

Himmelblau: \textrm{Memory} =  \frac{(x^2 + y - 11)^2 + (x + y^2 - 7)^2}{890} * 1024

x_{1}: \frac{a^2 + bc}{500*\log{d}} * 1024

x_{2} = \log{d} - \frac{e*h}{32}

y_{1} = \sin{\frac{a}{c} \pi } * cos(f*g \pi) - 2e

y_{2} =  \frac{\sqrt{be}}{f}
```
