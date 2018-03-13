# build stage
FROM golang:alpine AS build-env

# Create build dir
RUN mkdir -p /go/src/github.com/jackzampolin/balance-monitor

# Work out of build dir
WORKDIR /go/src/github.com/jackzampolin/balance-monitor

# Copy in source
COPY . .

# Get deps
RUN go get -u -t -f -v ./...

# Build app
RUN go build -o balance-monitor main.go

# Production Image
FROM alpine

COPY --from=build-env /go/src/github.com/jackzampolin/balance-monitor/balance-monitor /usr/bin/balance-monitor

COPY .balance-monitor.yaml /root/.balance-monitor.yaml

ENTRYPOINT ["/usr/bin/balance-monitor"]

CMD ["serve"]
