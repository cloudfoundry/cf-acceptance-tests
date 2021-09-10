# HTTP/2 Only APP

App that only supports cleartext HTTP/2 (h2c) traffic. Will 500 if it receives
non-HTTP/2 traffic.

## Running
1. `go build`
2. `PORT=8080 ./http2`
3. `curl localhost:8080 --http2`
