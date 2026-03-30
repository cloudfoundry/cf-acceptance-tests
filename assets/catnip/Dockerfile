FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src/app
COPY ./ ./

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/catnip /go/src/app/main.go

FROM alpine
COPY --from=builder /go/bin/catnip /go/bin/catnip

RUN apk add --no-cache ca-certificates bash curl lsb-release

ENV PORT=8080
CMD ["/go/bin/catnip"]
