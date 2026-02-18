FROM golang:1.25
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd/server
EXPOSE 8086

CMD ["./main"]