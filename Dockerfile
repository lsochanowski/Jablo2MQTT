ARG BUILD_FROM=golang:1.18
FROM $BUILD_FROM
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN /usr/local/bin/go mod download && go mod verify
COPY . .
RUN go build -v -o /usr/local/bin/app ./...
CMD ["app"]