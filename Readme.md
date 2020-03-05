```
$ ```
$ CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook .
$ docker-compose up
$ docker exec -it go_db_1 mysql -uaxiamed -paxiamed webhook
```