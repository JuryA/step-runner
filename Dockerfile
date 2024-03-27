FROM golang:1.21

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /step-runner

# This is necessary only during the transition period while step-runner
# is distributed as an image and not built into the CI environment.
RUN curl -fsSL https://get.docker.com | sh

CMD ["/step-runner"]
