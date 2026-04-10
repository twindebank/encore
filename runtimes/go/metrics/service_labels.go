//go:build encore_app

package metrics

// SetServiceLabels registers additional labels to be included with all
// metrics exported for the named service. This can be used to enrich built-in
// metrics (like e_requests_total) with custom metadata for alert routing or
// dashboard filtering.
//
// Labels are applied at export time to all metrics that include a service
// dimension. This function is safe to call concurrently. Subsequent calls
// for the same service replace previous labels.
//
// Example:
//
//	metrics.SetServiceLabels("myservice", map[string]string{
//	    "domain": "voice",
//	    "team":   "platform",
//	})
func SetServiceLabels(serviceName string, labels map[string]string) {
	Singleton.RegisterServiceLabels(serviceName, labels)
}
