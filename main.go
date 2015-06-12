// Copyright 2015 Seb
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type domoticzMetric struct {
	Id			uint32
	Type		string
	SType		string
    Name    	string
    Value   	float64
	Time        string
	Unit		string
}

var (
	listeningAddress = flag.String("web.listen-address", ":9103", "Address on which to expose metrics and web interface.")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "Path under which to expose Prometheus metrics.")
	domoticzPostPath = flag.String("web.domoticz-push-path", "/domoticz-post", "Path under which to accept POST requests from domoticz.")
	lastPush         = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "domoticz_last_push_timestamp_seconds",
			Help: "Unix timestamp of the last received domoticz metrics push in seconds.",
		},
	)
)

func metricName(m domoticzMetric) string {
	result := "domoticz"
	result += "_"
	result += fmt.Sprintf("%d",m.Id)
	result += "_"
	result += m.Type
        result += "_"
        result += m.SType
	return result
}

func metricHelp(m domoticzMetric) string {
	return fmt.Sprintf("Domoticz exporter: Type: '%s' Dstype: '%s' Dsname: '%s' Unit: '%s'", m.Type, m.SType, m.Name, m.Unit)
}

func metricType(dstype string) prometheus.ValueType {
	if dstype == "counter" || dstype == "derive" {
		return prometheus.CounterValue
	}
	return prometheus.GaugeValue
}

type domoticzSample struct {
	Id		uint32
	Name		string
	Labels 		map[string]string
	Help 		string
	Value         	float64
	DType		string
	Dstype		string
	Time           	float64
	Type		prometheus.ValueType
	Unit		string
	Expires 	time.Time
}

type domoticzCollector struct {
	samples map[uint32]*domoticzSample
	mu      *sync.Mutex
	ch      chan *domoticzSample
}

func newDomoticzCollector() *domoticzCollector {
		c := &domoticzCollector{
		ch:      make(chan *domoticzSample, 0),
		mu:      &sync.Mutex{},
		samples: map[uint32]*domoticzSample{},
	}
	go c.processSamples()
	return c
}

func (c *domoticzCollector) domoticzPost(w http.ResponseWriter, r *http.Request) {
	var postedMetric domoticzMetric
	err := json.NewDecoder(r.Body).Decode(&postedMetric)
	if err != nil {
		fmt.Printf("Error Decoding Json\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	now := time.Now()
	lastPush.Set(float64(now.UnixNano()) / 1e9)
	labels := prometheus.Labels{}
	c.ch <- &domoticzSample{
		Id:	 postedMetric.Id,
		Name:    metricName(postedMetric),
		Labels:  labels,
		Help:    metricHelp(postedMetric),
		Value:   postedMetric.Value,
		Type:    metricType(postedMetric.Name),
		Expires: now.Add(time.Duration(300) * time.Second * 2),
	}
}

func (c *domoticzCollector) processSamples() {
	ticker := time.NewTicker(time.Minute).C
	for {
		select {
		case sample := <-c.ch:
			c.mu.Lock()
			c.samples[sample.Id] = sample
			c.mu.Unlock()
		case <-ticker:
			// Garbage collect expired samples.
			now := time.Now()
			c.mu.Lock()
			for k, sample := range c.samples {
				if now.After(sample.Expires) {
					delete(c.samples, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

// Collect implements prometheus.Collector.
func (c domoticzCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- lastPush

	c.mu.Lock()
	samples := make([]*domoticzSample, 0, len(c.samples))
	for _, sample := range c.samples {
		samples = append(samples, sample)
	}
	c.mu.Unlock()

	now := time.Now()
	for _, sample := range samples {
		if now.After(sample.Expires) {
			continue
		}
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(sample.Name, sample.Help, []string{}, sample.Labels), sample.Type, sample.Value,
		)
	}
}

// Describe implements prometheus.Collector.
func (c domoticzCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- lastPush.Desc()
}

func main() {
	flag.Parse()
	http.Handle(*metricsPath, prometheus.Handler())
	c := newDomoticzCollector()
	http.HandleFunc(*domoticzPostPath, c.domoticzPost)
	prometheus.MustRegister(c)
	http.ListenAndServe(*listeningAddress, nil)
}
