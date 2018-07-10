# faas-idler

Idles functions after a period of inactivity

## Building

```
TAG=0.1.0 make build
```

## Usage

* Environmental variables:

gateway_url needs to be the URL of the faas-netes or faas-swarm service, which usually has no port exposed.

Try using the ClusterIP/Cluster Service instead and port 8080.

`gateway_url` - URL for faas-provider
`prometheus_host` - host for Prometheus
`prometheus_port` - port for Prometheus
`inactivity_duration` - i.e. `10m` (Golang duration)

* Command-line args

`-dry-run` - don't send scaling event 

How it works:

`gateway_function_invocation_total` is measured for activity over `duration` i.e. `1h` of inactivity (or no HTTP requests)

## Logs

```
kubectl logs -n openfaas -f deploy/faas-idler
```
