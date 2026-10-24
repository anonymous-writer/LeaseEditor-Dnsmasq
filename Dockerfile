FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . ./
COPY static ./static
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/leaseeditor-dnsmasq .

FROM alpine:3.21
RUN addgroup -S leaseeditor && adduser -S -G leaseeditor leaseeditor
WORKDIR /app
COPY --from=build /out/leaseeditor-dnsmasq /usr/local/bin/leaseeditor-dnsmasq
COPY config.yaml /app/config.yaml
EXPOSE 3009
USER leaseeditor
ENTRYPOINT ["/usr/local/bin/leaseeditor-dnsmasq", "--config", "/app/config.yaml"]
