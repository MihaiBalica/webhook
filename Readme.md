```
$ CGO_ENABLED=0 GOOS=linux go build -a -o webhook .
$ docker image build -t webhook:1.0.4 .
$ docker-compose up --scale app=4
```