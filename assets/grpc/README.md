# gRPC APP

App that serves gPRC traffic. Note that this will only work if all network hops
between the client and the app use HTTP/2. Configure your load balancers
appropriately.

## Running Locally
1. `go build`
2. `PORT=8080 ./grpc`
3. `grpcurl -vv -plaintext -import-path ./test -proto test.proto localhost:8080 test.Test.Run`

