FROM golang:1.20

RUN mkdir -p /usr/src/app

WORKDIR /usr/src/app

COPY server.go .

ENTRYPOINT ["go", "run", "server.go"]
