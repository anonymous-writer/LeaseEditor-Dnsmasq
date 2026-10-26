FROM golang:1.27-alpine AS build
WORKDIR /src
ARG APP_VERSION=dev
ARG APP_RELEASE_DATE=
ARG APP_CHANNEL=GoDev
ARG APP_UPDATED_AT=
COPY go.mod ./
COPY . ./
COPY static ./static
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildVersion=${APP_VERSION} -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildReleaseDate=${APP_RELEASE_DATE} -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildChannel=${APP_CHANNEL} -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildUpdatedAt=${APP_UPDATED_AT}" -o /out/leaseeditor-dnsmasq .

FROM alpine:3.21
RUN addgroup -S leaseeditor && adduser -S -G leaseeditor leaseeditor
WORKDIR /app
COPY --from=build /out/leaseeditor-dnsmasq /usr/local/bin/leaseeditor-dnsmasq
COPY config.yaml /app/config.yaml
EXPOSE 3009
USER leaseeditor
ENTRYPOINT ["/usr/local/bin/leaseeditor-dnsmasq", "--config", "/app/config.yaml"]
