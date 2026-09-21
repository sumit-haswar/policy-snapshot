FROM golang:1.24-alpine AS test
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./...

FROM test AS build
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/decision-gateway ./cmd/gateway

FROM alpine:3.21
RUN adduser -D -H -u 10001 app
COPY --from=build /out/decision-gateway /usr/local/bin/decision-gateway
USER app
EXPOSE 8080
ENTRYPOINT ["decision-gateway"]

