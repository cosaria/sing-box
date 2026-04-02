FROM golang:1.26-alpine AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${VERSION}" -o /sing-box ./cmd/sing-box/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /sing-box /usr/local/bin/sing-box

RUN mkdir -p /etc/sing-box

VOLUME /etc/sing-box
ENTRYPOINT ["sing-box"]
CMD ["serve", "--data-dir", "/etc/sing-box"]
