# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

# Copy manifests first so dependency downloads stay cached when only
# source files change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/archivyy .

FROM alpine:3.20
WORKDIR /app

RUN adduser -D -u 10001 app

COPY --from=build /out/archivyy /app/archivyy
COPY templates/ /app/templates/
COPY static/ /app/static/

USER app
EXPOSE 8080
CMD ["/app/archivyy"]
