FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go main_test.go ./
COPY static ./static
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/leaseeditor .

FROM alpine:3.21
RUN addgroup -S leaseeditor && adduser -S -G leaseeditor leaseeditor
WORKDIR /app
COPY --from=build /out/leaseeditor /app/leaseeditor
ENV HOST=0.0.0.0 PORT=3010 DNSMASQ_HOSTS_FILE=/data/dnsmasq.dhcphosts
EXPOSE 3010
USER leaseeditor
ENTRYPOINT ["/app/leaseeditor"]
