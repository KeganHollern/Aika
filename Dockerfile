FROM golang:1.19-alpine as builder
# Install required C libraries and tools
RUN apk add --no-cache git gcc musl-dev pkgconf opus-dev alsa-lib alsa-lib-dev
# Set PKG_CONFIG_PATH for alsa
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .
FROM alpine:latest as runtime
RUN apk --no-cache add ca-certificates ffmpeg opus
WORKDIR /aika/
COPY --from=builder /app/main .
CMD ["./main"]  