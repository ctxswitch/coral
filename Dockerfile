FROM golang:1.22 AS build
WORKDIR /app
RUN apt-get update && apt-get install -y libgpgme-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux make build

FROM debian:bookworm-slim AS base
RUN apt-get update && apt-get install -y ca-certificates

FROM scratch
ENV NODE_NAME=localhost
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/bin/coral /coral

CMD ["/coral", "controller", "--log-level=6"]