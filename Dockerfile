FROM gcr.io/distroless/static

COPY nanomdm-linux-amd64 /nanomdm
COPY nano2nano-linux-amd64 /nano2nano

EXPOSE 9000

VOLUME ["/db"]

ENTRYPOINT ["/nanomdm"]
