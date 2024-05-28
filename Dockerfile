FROM golang:1.22.3-alpine3.18 AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -v -o main .

FROM alpine:3.18 AS runner

# Set default timezone
RUN apk add --no-cache tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app

COPY --from=builder /app/main .

ENTRYPOINT ["./main"]