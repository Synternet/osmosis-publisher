# Build container
FROM golang:1.21 AS build

COPY go.mod go.sum /home/src/
WORKDIR /home/src
RUN go mod download
COPY . /home/src

RUN make build

# Runtime container
FROM debian:bookworm-slim AS osmosis-publisher

ENV TZ=UTC

WORKDIR /home/app
COPY --from=build /home/src/build/osmosis-publisher .
COPY --from=build /home/src/build/libwasmvm.x86_64.so .
COPY --from=build /home/src/docker/entrypoint.sh .
ENV LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:/home/app"
RUN chmod +x /home/app/entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
