FROM golang:1.21.1-alpine as builder
WORKDIR /app

RUN apk add --no-cache git gcc musl-dev pkgconf opus-dev alsa-lib alsa-lib-dev
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest as runtime
WORKDIR /aika

RUN apk --no-cache add ca-certificates ffmpeg opus
COPY --from=builder /app/main .

CMD ["./main"]  