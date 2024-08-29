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

ARG COLOR
ENV COLOR=${COLOR}
ARG ERROR_RATE
ENV ERROR_RATE=${ERROR_RATE}
ARG LATENCY
ENV LATENCY=${LATENCY}

ENTRYPOINT [ "/rollouts-demo" ]
