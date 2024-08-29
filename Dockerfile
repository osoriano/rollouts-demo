FROM golang:1.23 as build
WORKDIR /go/src/app
COPY main.go .
RUN go mod init github.com/osoriano/rollouts-demo && \
  [ -z "$(go fmt)" ] && \
  go vet && \
  CGO_ENABLED=0 go build

FROM scratch
COPY static/* ./
COPY --from=build /go/src/app/rollouts-demo /rollouts-demo

ENTRYPOINT [ "/rollouts-demo" ]
