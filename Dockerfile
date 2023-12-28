FROM golang:1.21 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /step-runner

FROM alpine

COPY --from=build /step-runner /step-runner

RUN apk add git bash neofetch

CMD ["/step-runner"]

EXPOSE 8765
