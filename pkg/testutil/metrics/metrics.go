package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Value returns the value of the counter or gauge matching name and labels.
// An absent metric series is treated as zero.
func Value(gatherer prometheus.Gatherer, name string, labels prometheus.Labels) (float64, error) {
	families, err := gatherer.Gather()
	if err != nil {
		return 0, fmt.Errorf("gather metrics: %w", err)
	}

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.GetMetric() {
			matches := true

			for labelName, expected := range labels {
				found := false
				for _, pair := range metric.GetLabel() {
					if pair.GetName() == labelName && pair.GetValue() == expected {
						found = true
						break
					}
				}

				if !found {
					matches = false
					break
				}
			}

			if !matches {
				continue
			}

			switch {
			case metric.Counter != nil:
				return metric.GetCounter().GetValue(), nil
			case metric.Gauge != nil:
				return metric.GetGauge().GetValue(), nil
			default:
				return 0, fmt.Errorf("metric %q is not a counter or gauge", name)
			}
		}
	}

	return 0, nil
}
