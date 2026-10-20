FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" -o /skypod .

FROM scratch
COPY --from=build /skypod /skypod
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/skypod", "-addr", ":8080"]
