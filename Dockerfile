FROM golang:alpine as build

RUN apk update && apk upgrade && apk add --no-cache bash git openssh gcc musl-dev
ENV GOROOT=/usr/local/go
COPY . /usr/local/go/src/github.com/mit-dci/go-bverify
WORKDIR /usr/local/go/src/github.com/mit-dci/go-bverify
RUN go get ./...
WORKDIR /usr/local/go/src/github.com/mit-dci/go-bverify/cmd/server
RUN go build

FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
RUN cd /app
COPY --from=build /usr/local/go/src/github.com/mit-dci/go-bverify/cmd/server/server /app/bin/server

EXPOSE 8001

CMD ["bin/server"]