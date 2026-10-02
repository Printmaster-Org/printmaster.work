FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/printmaster-site .

FROM alpine:3.21
RUN addgroup -S -g 10001 web && adduser -S -D -H -u 10001 -G web web
COPY --from=build /out/printmaster-site /usr/local/bin/printmaster-site
USER 10001:10001
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/usr/local/bin/printmaster-site"]