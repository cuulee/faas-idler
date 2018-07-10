package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	providerTypes "github.com/openfaas/faas-provider/types"
	"github.com/openfaas/faas/gateway/metrics"
	"github.com/openfaas/faas/gateway/requests"
)

var dryRun bool

func main() {

	flag.BoolVar(&dryRun, "dry-run", false, "use dry-run for scaling events")
	flag.Parse()

	reconcileInterval := time.Second * 30

	client := &http.Client{}
	gatewayURL := os.Getenv("gateway_url")
	prometheusHost := os.Getenv("prometheus_host")
	inactivityDurationVal := os.Getenv("inactivity_duration")

	if len(inactivityDurationVal) == 0 {
		inactivityDurationVal = "10m"
	}

	inactivityDuration, _ := time.ParseDuration(inactivityDurationVal)

	prometheusPortVal := os.Getenv("prometheus_port")

	var prometheusPort int
	if len(prometheusPortVal) > 0 {
		var err error
		prometheusPort, err = strconv.Atoi(prometheusPortVal)
		if err != nil {
			log.Panicln(err)
		}
	}

	fmt.Printf(`dry_run: %t
gateway_url: %s
inactivity_duration: %s `, dryRun, gatewayURL, inactivityDuration)

	if len(gatewayURL) == 0 {
		fmt.Println("gateway_url (faas-netes/faas-swarm) is required.")
		os.Exit(1)
	}

	for {

		reconcile(client, gatewayURL, prometheusHost, prometheusPort, inactivityDuration)
		time.Sleep(reconcileInterval)
		fmt.Printf("\n")
	}
}

func buildMetricsMap(client *http.Client, functions []requests.Function, prometheusHost string, prometheusPort int, inactivityDuration time.Duration) map[string]float64 {
	query := metrics.NewPrometheusQuery(prometheusHost, prometheusPort, client)
	metrics := make(map[string]float64)

	duration := fmt.Sprintf("%dm", int(inactivityDuration.Minutes()))
	// duration := "5m"

	for _, function := range functions {
		querySt := url.QueryEscape(`sum(rate(gateway_function_invocation_total{function_name="` + function.Name + `", code=~".*"}[` + duration + `])) by (code, function_name)`)
		// fmt.Println(function.Name)
		res, err := query.Fetch(querySt)
		if err != nil {
			log.Println(err)
			continue
		}

		if len(res.Data.Result) > 0 {
			for _, v := range res.Data.Result {
				fmt.Println(v)
				if v.Metric.FunctionName == function.Name {
					metricValue := v.Value[1]
					switch metricValue.(type) {
					case string:
						// log.Println("String")
						f, strconvErr := strconv.ParseFloat(metricValue.(string), 64)
						if strconvErr != nil {
							log.Printf("Unable to convert value for metric: %s\n", strconvErr)
							continue
						}
						metrics[function.Name] = f
						break
					}
				}
			}

		}

	}

	return metrics
}

func reconcile(client *http.Client, gatewayURL, prometheusHost string, prometheusPort int, inactivityDuration time.Duration) {
	functions, err := queryFunctions(client, gatewayURL)

	if err != nil {
		log.Println(err)
		return
	}

	metrics := buildMetricsMap(client, functions, prometheusHost, prometheusPort, inactivityDuration)

	for _, fn := range functions {
		if v, found := metrics[fn.Name]; found {
			if v == float64(0) {
				fmt.Printf("%s\tidle\n", fn.Name)

				if val, _ := getReplicas(client, gatewayURL, fn.Name); val != nil && val.AvailableReplicas > 0 {
					sendScaleEvent(client, gatewayURL, fn.Name, uint64(0))
				}

			} else {
				fmt.Printf("%s\tactive: %f\n", fn.Name, v)
			}
		}
	}
}

func getReplicas(client *http.Client, gatewayURL string, name string) (*requests.Function, error) {
	item := &requests.Function{}
	var err error

	req, _ := http.NewRequest(http.MethodGet, gatewayURL+"system/function/"+name, nil)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	bytesOut, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(bytesOut, &item)

	return item, err
}

func queryFunctions(client *http.Client, gatewayURL string) ([]requests.Function, error) {
	list := []requests.Function{}
	var err error

	req, _ := http.NewRequest(http.MethodGet, gatewayURL+"system/functions", nil)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	bytesOut, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(bytesOut, &list)

	return list, err
}

func sendScaleEvent(client *http.Client, gatewayURL string, name string, replicas uint64) {
	if dryRun {
		fmt.Printf("dry-run: Scaling %s to %d replicas\n", name, replicas)
		return
	}

	scaleReq := providerTypes.ScaleServiceRequest{
		ServiceName: name,
		Replicas:    replicas,
	}

	var err error

	bodyBytes, _ := json.Marshal(scaleReq)
	bodyReader := bytes.NewReader(bodyBytes)

	req, _ := http.NewRequest(http.MethodPost, gatewayURL+"system/scale-function/"+name, bodyReader)

	res, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Scale", name, res.StatusCode, replicas)

	if res.Body != nil {
		defer res.Body.Close()
	}
}
