package aggregator

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/suchpuppet/kubectl-mc/pkg/executor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TableAggregator formats multi-cluster results as a kubectl-style table
type TableAggregator struct {
	writer io.Writer
}

// ItemWithCluster represents a Kubernetes resource with its cluster information
type ItemWithCluster struct {
	Item    unstructured.Unstructured
	Cluster string
}

// NewTableAggregator creates a new table aggregator
func NewTableAggregator(writer io.Writer) *TableAggregator {
	return &TableAggregator{
		writer: writer,
	}
}

// AggregateGetResults aggregates and formats get results across clusters
func (a *TableAggregator) AggregateGetResults(results *executor.AggregatedResults, resourceType string) error {
	// Collect all items with cluster information
	var allItems []ItemWithCluster

	for _, result := range results.Results {
		if !result.Success {
			continue
		}
		for _, item := range result.Items {
			allItems = append(allItems, ItemWithCluster{
				Item:    item,
				Cluster: result.ClusterName,
			})
		}
	}

	if len(allItems) == 0 {
		fmt.Fprintln(a.writer, "No resources found")
		return nil
	}

	// Sort by cluster, then namespace, then name
	sort.Slice(allItems, func(i, j int) bool {
		if allItems[i].Cluster != allItems[j].Cluster {
			return allItems[i].Cluster < allItems[j].Cluster
		}
		nsI, _, _ := unstructured.NestedString(allItems[i].Item.Object, "metadata", "namespace")
		nsJ, _, _ := unstructured.NestedString(allItems[j].Item.Object, "metadata", "namespace")
		if nsI != nsJ {
			return nsI < nsJ
		}
		nameI, _, _ := unstructured.NestedString(allItems[i].Item.Object, "metadata", "name")
		nameJ, _, _ := unstructured.NestedString(allItems[j].Item.Object, "metadata", "name")
		return nameI < nameJ
	})

	// Format based on resource type
	switch strings.ToLower(resourceType) {
	case "pod", "pods":
		return a.formatPods(allItems)
	case "deployment", "deployments":
		return a.formatDeployments(allItems)
	case "service", "services":
		return a.formatServices(allItems)
	default:
		return a.formatGeneric(allItems)
	}
}

// formatPods formats pod resources
func (a *TableAggregator) formatPods(items []ItemWithCluster) error {
	// Header
	fmt.Fprintln(a.writer, "NAMESPACE\tNAME\tCLUSTER\tREADY\tSTATUS\tRESTARTS\tAGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")

		// Get pod status
		phase, _, _ := unstructured.NestedString(item.Item.Object, "status", "phase")

		// Get container statuses for ready count
		ready := "0/0"
		restarts := int64(0)
		if containerStatuses, found, _ := unstructured.NestedSlice(item.Item.Object, "status", "containerStatuses"); found {
			total := len(containerStatuses)
			readyCount := 0
			for _, cs := range containerStatuses {
				csMap, ok := cs.(map[string]interface{})
				if !ok {
					continue
				}
				if isReady, found, _ := unstructured.NestedBool(csMap, "ready"); found && isReady {
					readyCount++
				}
				if count, found, _ := unstructured.NestedInt64(csMap, "restartCount"); found {
					restarts += count
				}
			}
			ready = fmt.Sprintf("%d/%d", readyCount, total)
		}

		// Calculate age (simplified - just show creation time for now)
		age := "<unknown>"
		if creationTime, found, _ := unstructured.NestedString(item.Item.Object, "metadata", "creationTimestamp"); found && creationTime != "" {
			// Simplified age display
			age = "---" // TODO: Calculate actual age
		}

		fmt.Fprintf(a.writer, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			ns, name, item.Cluster, ready, phase, restarts, age)
	}

	return nil
}

// formatDeployments formats deployment resources
func (a *TableAggregator) formatDeployments(items []ItemWithCluster) error {
	// Header
	fmt.Fprintln(a.writer, "NAMESPACE\tNAME\tCLUSTER\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")

		replicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "replicas")
		readyReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "readyReplicas")
		updatedReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "updatedReplicas")
		availableReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "availableReplicas")

		ready := fmt.Sprintf("%d/%d", readyReplicas, replicas)

		fmt.Fprintf(a.writer, "%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			ns, name, item.Cluster, ready, updatedReplicas, availableReplicas, "---")
	}

	return nil
}

// formatServices formats service resources
func (a *TableAggregator) formatServices(items []ItemWithCluster) error {
	// Header
	fmt.Fprintln(a.writer, "NAMESPACE\tNAME\tCLUSTER\tTYPE\tCLUSTER-IP\tEXTERNAL-IP\tPORT(S)\tAGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")

		svcType, _, _ := unstructured.NestedString(item.Item.Object, "spec", "type")
		clusterIP, _, _ := unstructured.NestedString(item.Item.Object, "spec", "clusterIP")
		externalIP := "<none>"

		// Get ports
		ports := "<none>"
		if portsSlice, found, _ := unstructured.NestedSlice(item.Item.Object, "spec", "ports"); found && len(portsSlice) > 0 {
			var portStrs []string
			for _, p := range portsSlice {
				pMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}
				port, _, _ := unstructured.NestedInt64(pMap, "port")
				protocol, _, _ := unstructured.NestedString(pMap, "protocol")
				portStrs = append(portStrs, fmt.Sprintf("%d/%s", port, protocol))
			}
			if len(portStrs) > 0 {
				ports = strings.Join(portStrs, ",")
			}
		}

		fmt.Fprintf(a.writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ns, name, item.Cluster, svcType, clusterIP, externalIP, ports, "---")
	}

	return nil
}

// formatGeneric formats any resource type in a generic way
func (a *TableAggregator) formatGeneric(items []ItemWithCluster) error {
	// Header
	fmt.Fprintln(a.writer, "NAMESPACE\tNAME\tCLUSTER\tKIND\tAGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		kind := item.Item.GetKind()

		if ns == "" {
			ns = "<none>"
		}

		fmt.Fprintf(a.writer, "%s\t%s\t%s\t%s\t%s\n",
			ns, name, item.Cluster, kind, "---")
	}

	return nil
}
