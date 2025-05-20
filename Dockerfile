FROM gcr.io/distroless/static

ARG TARGETOS TARGETARCH

COPY nanomdm-$TARGETOS-$TARGETARCH /app/nanomdm
COPY nano2nano-$TARGETOS-$TARGETARCH /app/nano2nano

EXPOSE 9000

VOLUME ["/app/dbkv", "/app/db"]

WORKDIR /app

ENTRYPOINT ["/app/nanomdm"]
