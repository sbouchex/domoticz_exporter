# Domoticz Exporter

An exporter for [Domoticz](https://www.domoticz.com/). It accepts domoticz metrics
in JSON format via HTTP POST and transforms and exposes them for consumption by Prometheus.

This exporter is useful for exporting metrics from existing domoticz setups, as
well as for metrics which are not covered by the core Prometheus exporters such
as the [Node Exporter](https://github.com/prometheus/node_exporter).

Domoticz configuration must be as followed : Http push must be enabled and confured as followed
* URL : http://<url>:9103/domoticz-post
* Method : POST
* Authentication : None
* Data : {"Id":%V,"Type":"%T0","SType":"%T1","Name":"%n", "Value":%v, "Time":"%t4","Unit":"%u"}
