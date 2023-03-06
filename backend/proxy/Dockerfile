FROM golang:1.20

RUN mkdir -p /usr/src/app

WORKDIR /usr/src/app

COPY proxy.go .

ENTRYPOINT ["go", "run", "proxy.go"]
