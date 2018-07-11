# faas-idler

Scale functions to zero replicas after a period of inactivity

Premise: functions (Deployments) can be scaled to 0/0 replicas from 1/1 or N/N replicas when they are not receiving traffic. Traffic is observed from Prometheus metrics collected in the OpenFaaS API Gateway.

Scaling to zero requires an "un-idler" or a blocking HTTP proxy which can reverse the process when incoming requests attempt to access a given function. This is done through the OpenFaaS API Gateway through which every incoming call passes.

faas-idler is implemented as a controller which polls Prometheus metrics on a regular basis and tries to reconcile a desired condition - i.e. zero replicas -> scale down API call.

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

The `reconcileInterval` is hard-coded to run every 30s.

* Command-line args

`-dry-run` - don't send scaling event 

How it works:

`gateway_function_invocation_total` is measured for activity over `duration` i.e. `1h` of inactivity (or no HTTP requests)

## Logs

```
kubectl logs -n openfaas -f deploy/faas-idler
```
