FROM golang
WORKDIR /go/src/app
COPY ./ ./

RUN go build -o /go/bin/catnip /go/src/app/main.go

ENV PORT=8080
CMD ["/go/bin/catnip"]
