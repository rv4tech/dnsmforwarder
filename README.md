# dnsmforwarder
Simple DNS proxy (multi origins to upstream forwarder)

```bash
./dnsmforwarder &

curl -vvv -X PUT -H 'Content-type: application/json' -d '{"ip": "127.0.0.1", "upstream": "8.8.8.8:53"}' http://127.0.0.1:10080/api/v1/origins
curl -vvv -X PUT -H 'Content-type: application/json' -d '{"upstream": "8.8.8.8:53"}' http://127.0.0.1:10080/api/v1/upstreams

dig -p 10053 @127.0.0.1 google.com
```
