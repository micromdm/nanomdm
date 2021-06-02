FROM gcr.io/distroless/static

COPY nanomdm-linux-amd64 /nanomdm

EXPOSE 9000

VOLUME ["/db"]

ENTRYPOINT ["/nanomdm"]
