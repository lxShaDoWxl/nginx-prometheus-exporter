package collector

import (
	unitclient "github.com/nginxinc/nginx-prometheus-exporter/client/unit"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// NginxUnitCollector collects NGINX metrics. It implements prometheus.Collector interface.
type NginxUnitCollector struct {
	nginxClient        *unitclient.NginxClient
	metrics            map[string]*prometheus.Desc
	applicationMetrics map[string]*prometheus.Desc
	upMetric           prometheus.Gauge
	mutex              sync.Mutex
	logger             log.Logger
}

// NewNginxUnitCollector creates an NewNginxUnitCollector.
func NewNginxUnitCollector(nginxClient *unitclient.NginxClient, namespace string, constLabels map[string]string, logger log.Logger) *NginxUnitCollector {
	return &NginxUnitCollector{
		nginxClient: nginxClient,
		logger:      logger,
		metrics: map[string]*prometheus.Desc{
			"connections_accepted": newGlobalMetric(namespace, "connections_accepted", "Accepted client connections", constLabels),
			"connections_active":   newGlobalMetric(namespace, "connections_active", "Active client connections", constLabels),
			"connections_idle":     newGlobalMetric(namespace, "connections_idle", "Idle client connections", constLabels),
			"connections_closed":   newGlobalMetric(namespace, "connections_closed", "Closed client connections", constLabels),
			"http_requests_total":  newGlobalMetric(namespace, "http_requests_total", "Total http requests", constLabels),
		},
		applicationMetrics: map[string]*prometheus.Desc{
			"processes_running":  newApplicationServerMetric(namespace, "processes_running", "Application processes running", []string{}, constLabels),
			"processes_starting": newApplicationServerMetric(namespace, "processes_starting", "Application processes starting", []string{}, constLabels),
			"processes_idle":     newApplicationServerMetric(namespace, "processes_idle", "Application processes idle", []string{}, constLabels),
			"requests_active":    newApplicationServerMetric(namespace, "requests_active", "Active requests", []string{}, constLabels),
		},
		upMetric: newUpMetric(namespace, constLabels),
	}
}

// Describe sends the super-set of all possible descriptors of NGINX metrics
// to the provided channel.
func (c *NginxUnitCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upMetric.Desc()

	for _, m := range c.metrics {
		ch <- m
	}
	for _, m := range c.applicationMetrics {
		ch <- m
	}
}

// Collect fetches metrics from NGINX and sends them to the provided channel.
func (c *NginxUnitCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects
	defer c.mutex.Unlock()

	stats, err := c.nginxClient.GetStatus()
	if err != nil {
		c.upMetric.Set(nginxDown)
		ch <- c.upMetric
		level.Error(c.logger).Log("msg", "Error getting stats", "error", err.Error())
		return
	}

	c.upMetric.Set(nginxUp)
	ch <- c.upMetric

	ch <- prometheus.MustNewConstMetric(c.metrics["connections_accepted"],
		prometheus.CounterValue, float64(stats.Connections.Accepted))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_active"],
		prometheus.GaugeValue, float64(stats.Connections.Active))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_idle"],
		prometheus.GaugeValue, float64(stats.Connections.Idle))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_closed"],
		prometheus.CounterValue, float64(stats.Connections.Closed))
	ch <- prometheus.MustNewConstMetric(c.metrics["http_requests_total"],
		prometheus.CounterValue, float64(stats.Requests.Total))
	for s, application := range stats.Applications {
		ch <- prometheus.MustNewConstMetric(c.applicationMetrics["processes_running"],
			prometheus.GaugeValue, float64(application.Processes.Running), s)
		ch <- prometheus.MustNewConstMetric(c.applicationMetrics["processes_starting"],
			prometheus.GaugeValue, float64(application.Processes.Starting), s)
		ch <- prometheus.MustNewConstMetric(c.applicationMetrics["processes_idle"],
			prometheus.GaugeValue, float64(application.Processes.Idle), s)
		ch <- prometheus.MustNewConstMetric(c.applicationMetrics["requests_active"],
			prometheus.GaugeValue, float64(application.Requests.Active), s)
	}

}

func newApplicationServerMetric(namespace string, metricName string, docString string, variableLabelNames []string, constLabels prometheus.Labels) *prometheus.Desc {
	labels := []string{"application"}
	labels = append(labels, variableLabelNames...)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "applications", metricName), docString, labels, constLabels)
}
