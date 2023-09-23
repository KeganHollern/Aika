FROM golang:1.19 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
FROM alpine:latest as runtime
RUN apk --no-cache add ca-certificates
WORKDIR /aika/
COPY --from=builder /app/main .

# TODO: configure ffmpeg and opus libraries

CMD ["./main"]  