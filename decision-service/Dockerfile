FROM golang:1.24-alpine AS test
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./...

FROM test AS build
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/decision-service ./cmd/decisionservice

FROM alpine:3.21
RUN adduser -D -H -u 10001 app
COPY --from=build /out/decision-service /usr/local/bin/decision-service
USER app
EXPOSE 8081
ENTRYPOINT ["decision-service"]

