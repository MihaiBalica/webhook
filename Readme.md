```
$ CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook .
$ docker image build -t webhook:1.0.1 .
$ docker-compose up
```