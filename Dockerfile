FROM golang:1.16 AS builder
ENV GO111MODULE=on
ADD ./ /data/work/raft-kv/
WORKDIR /data/work/raft-kv/
RUN go mod download
RUN go build -o /data/app/raft ./main.go

FROM golang:1.16
COPY --from=builder /data/app/raft /app/
CMD /app/raft
