FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# goreleaser copies the pre-built binary into the build context
COPY sing-box-linux-* /usr/local/bin/sing-box

RUN chmod +x /usr/local/bin/sing-box && mkdir -p /etc/sing-box

VOLUME /etc/sing-box
ENTRYPOINT ["sing-box"]
CMD ["serve", "--data-dir", "/etc/sing-box"]
