
```
$ docker-compose build --parallel
$ docker-compose up --scale app=4

$ more input.json 
{
  "id": 12345,
  "client_id": 9876543210,
  "ip_address": "192.168.1.52",
  "user_name": "ahmed",
  "customer_name": "Jamal",
  "software_name": "WinRar"
}
$ curl -X POST http://localhost:8080/initializeDataBase
$ curl -X POST -H 'Content-Type: application/json' -d @input.json http://localhost:8080/
$ curl -X GET http://localhost:8080/12345?

```