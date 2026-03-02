package k8

import (
	"fmt"
	"time"
)

// extractCondition pulls Ready status, reason, and last transition time from a
// FluxCD object's .status.conditions array (standard across all Flux CRDs).
func extractCondition(obj map[string]interface{}) (ready, status, lastSeen string) {
	ready, status, lastSeen = "Unknown", "Unknown", "Unknown"
	statusBlock, ok := obj["status"].(map[string]interface{})
	if !ok {
		return
	}
	conditions, ok := statusBlock["conditions"].([]interface{})
	if !ok {
		return
	}
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Ready" {
			if v, ok := cond["status"].(string); ok {
				ready = v
			}
			if v, ok := cond["reason"].(string); ok {
				status = v
			}
			if v, ok := cond["lastTransitionTime"].(string); ok {
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					lastSeen = formatAgo(time.Since(t))
				}
			}
			return
		}
	}
	return
}

func formatAgo(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
