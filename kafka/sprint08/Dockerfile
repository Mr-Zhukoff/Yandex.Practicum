FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_PATH
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/service ./${SERVICE_PATH}

FROM alpine:3.20

RUN adduser -D -H appuser
USER appuser

COPY --from=build /out/service /usr/local/bin/service

ENTRYPOINT ["/usr/local/bin/service"]
