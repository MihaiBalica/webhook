FROM golang:latest
ENV PORT=8080
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /go
COPY ./http-server.go /go

RUN apt -y update && apt -y install git
RUN go get github.com/go-sql-driver/mysql

RUN cd /go &&  go build -a -o webhook .

EXPOSE 8080

CMD ["/go/webhook"]
