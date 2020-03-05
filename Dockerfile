FROM golang:1.14.0-alpine
ENV PORT=8090
WORKDIR /go
RUN go get -d -v github.com/go-sql-driver/mysql
COPY http-server.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /go

COPY --from=0 /go    .

CMD ["./webhook"]
