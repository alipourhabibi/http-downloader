FROM golang:alpine
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-s -w' -o httpdl

FROM alpine:latest
COPY --from=0 /app/httpdl /bin/httpdl
