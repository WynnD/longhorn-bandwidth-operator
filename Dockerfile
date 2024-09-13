# Build the application from source
FROM golang:1.22 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM scratch

COPY --from=build-stage /app/app /app

ENTRYPOINT ["/app"]
