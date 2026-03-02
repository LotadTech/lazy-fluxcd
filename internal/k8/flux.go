package k8

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// FluxResource identifies a FluxCD CRD group/version/resource.
var fluxResources = map[string]schema.GroupVersionResource{
	// Kustomize controller
	"Kustomizations": {Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Resource: "kustomizations"},
	// Helm controller
	"Helm Releases": {Group: "helm.toolkit.fluxcd.io", Version: "v2", Resource: "helmreleases"},
	// Source controller
	"Git Repositories":  {Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "gitrepositories"},
	"OCI Repositories":  {Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "ocirepositories"},
	"Helm Repositories": {Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "helmrepositories"},
	"Helm Charts":       {Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "helmcharts"},
	"Buckets":           {Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "buckets"},
	// Notification controller
	"Alerts":    {Group: "notification.toolkit.fluxcd.io", Version: "v1beta3", Resource: "alerts"},
	"Providers": {Group: "notification.toolkit.fluxcd.io", Version: "v1beta3", Resource: "providers"},
	"Receivers": {Group: "notification.toolkit.fluxcd.io", Version: "v1", Resource: "receivers"},

}

// Row is a single display row: name, ready, status, last reconciled.
type Row [4]string

// FetchFluxResources returns rows for the given category across all namespaces.
func FetchFluxResources(client dynamic.Interface, category string) ([]Row, error) {
	gvr, ok := fluxResources[category]
	if !ok {
		return nil, fmt.Errorf("unknown category: %s", category)
	}

	list, err := client.Resource(gvr).Namespace("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var rows []Row
	for _, item := range list.Items {
		name := item.GetName()
		ready, status, lastSeen := extractCondition(item.Object)
		rows = append(rows, Row{name, ready, status, lastSeen})
	}
	return rows, nil
}
