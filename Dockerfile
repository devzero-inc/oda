# This is a two phase docker build. It uses stage 1 to compile the binary,
# then in stage 2 bundles just the config and binary into a docker container.

# Stage 1 -> Build project
ARG GOVERSION=1.21.4-alpine3.18
FROM golang:${GOVERSION} AS builder

RUN apk update && \
    apk add ca-certificates git curl gcc musl-dev build-base autoconf automake libtool make

RUN update-ca-certificates

WORKDIR /src

# Copy project files
COPY . .

# Grab all dependencies
RUN go get -t -v ./...

# Use the make command here for consistency, to avoid docker and dev binary build drift.
RUN make build

# Stage 2 -> Serve the application
FROM scratch as release

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /src/oda ./

ENTRYPOINT ["/oda"]
