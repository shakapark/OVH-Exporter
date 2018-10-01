FROM golang:1.10 AS build
ADD src /go/src/OVH-Exporter/src
WORKDIR /go/src/OVH-Exporter/src
RUN go get -d -v
RUN CGO_ENABLED=0 go build -o ovh-exporter

FROM alpine
RUN apk --no-cache add ca-certificates && update-ca-certificates
WORKDIR /app
COPY --from=build /go/src/OVH-Exporter/src/ovh-exporter /app/
ADD ovh.yml /app/config/
ENTRYPOINT [ "/app/ovh-exporter","--config.file=/app/config/ovh.yml" ]