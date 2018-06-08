FROM golang:1.10-alpine AS builder

RUN apk update && apk add curl git
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

ADD ./src /go/src/github.com/alexander-akhmetov/raft-example/src
ADD ./vendor /go/src/github.com/alexander-akhmetov/raft-example/vendor
ADD Gopkg.toml /go/src/github.com/alexander-akhmetov/raft-example/
ADD Gopkg.lock /go/src/github.com/alexander-akhmetov/raft-example/

WORKDIR /go/src/github.com/alexander-akhmetov/raft-example/

RUN dep ensure -vendor-only
RUN go build -o /tmp/raft ./src/*.go


FROM golang:1.10-alpine
COPY --from=builder /tmp/raft /app/
CMD /app/raft
