FROM golang as pora
WORKDIR /go/src/app
COPY * .
RUN go build -o /pora /go/src/app/server.go

FROM ubuntu
LABEL org.cloudfoundry.pora.dockerfile.url="https://github.com/cloudfoundry/cf-acceptance-tests/blob/main/assets/pora/Dockerfile"
LABEL org.cloudfoundry.pora.notes.md="Used by code.cloudfoundry.org/cf-volume-services-acceptance-tests\
"
COPY --from=pora /pora /usr/local/bin/pora
ENTRYPOINT /usr/local/bin/pora
ENV PORT=8080
EXPOSE 8080
