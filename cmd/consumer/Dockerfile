FROM golang:latest
WORKDIR /app

COPY go.mod go.sum ./
COPY .env ./
RUN go mod download
RUN wget https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-v0.6.1.tar.gz \
    && rm dockerize-linux-amd64-v0.6.1.tar.gz

COPY ./cmd/consumer .
COPY internal/consumer/ internal/consumer/
COPY internal/utils/ internal/utils/
COPY internal/websocket/ internal/websocket/

RUN go build -o main .
CMD dockerize -wait tcp://kafka:9092 -timeout 90s go run main.go
