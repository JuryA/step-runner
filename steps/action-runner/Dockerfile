FROM golang:1.21 as build

WORKDIR /app

COPY . ./
RUN go mod download
RUN go build -o /usr/bin/action-runner .

FROM debian:latest

COPY --from=build /usr/bin/action-runner /usr/bin/

RUN apt-get update && apt-get install -y ca-certificates

ENTRYPOINT ["/usr/bin/action-runner"]
