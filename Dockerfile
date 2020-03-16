FROM alpine:latest
ENV PORT=8080

WORKDIR /go
COPY ./webhook .

EXPOSE 8080

CMD ["./webhook"]
