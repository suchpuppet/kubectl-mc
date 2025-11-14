package aggregator

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/suchpuppet/kubectl-mc/pkg/executor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	noneValue = "<none>"
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

// podColumnWidths holds column widths for pod table
type podColumnWidths struct {
	namespace int
	name      int
	cluster   int
	ready     int
	status    int
	restarts  int
}

// deploymentColumnWidths holds column widths for deployment table
type deploymentColumnWidths struct {
	namespace int
	name      int
	cluster   int
	ready     int
	upToDate  int
	available int
}

// serviceColumnWidths holds column widths for service table
type serviceColumnWidths struct {
	namespace  int
	name       int
	cluster    int
	svcType    int
	clusterIP  int
	externalIP int
	ports      int
}

// genericColumnWidths holds column widths for generic table
type genericColumnWidths struct {
	namespace int
	name      int
	cluster   int
	kind      int
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
	// Calculate column widths dynamically
	widths := a.calculatePodColumnWidths(items)

	// Header
	fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*s %-*s %s\n",
		widths.namespace, "NAMESPACE",
		widths.name, "NAME",
		widths.cluster, "CLUSTER",
		widths.ready, "READY",
		widths.status, "STATUS",
		widths.restarts, "RESTARTS",
		"AGE")

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

		// Calculate age
		age := calculateAge(item.Item)

		fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*s %-*d %s\n",
			widths.namespace, ns,
			widths.name, name,
			widths.cluster, item.Cluster,
			widths.ready, ready,
			widths.status, phase,
			widths.restarts, restarts,
			age)
	}

	return nil
}

// calculatePodColumnWidths calculates optimal column widths for pod table
func (a *TableAggregator) calculatePodColumnWidths(items []ItemWithCluster) podColumnWidths {
	widths := podColumnWidths{
		namespace: len("NAMESPACE"),
		name:      len("NAME"),
		cluster:   len("CLUSTER"),
		ready:     len("READY"),
		status:    len("STATUS"),
		restarts:  len("RESTARTS"),
	}

	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		phase, _, _ := unstructured.NestedString(item.Item.Object, "status", "phase")

		if len(ns) > widths.namespace {
			widths.namespace = len(ns)
		}
		if len(name) > widths.name {
			widths.name = len(name)
		}
		if len(item.Cluster) > widths.cluster {
			widths.cluster = len(item.Cluster)
		}

		// Ready format is "X/Y"
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
			}
			readyStr := fmt.Sprintf("%d/%d", readyCount, total)
			if len(readyStr) > widths.ready {
				widths.ready = len(readyStr)
			}
		}

		if len(phase) > widths.status {
			widths.status = len(phase)
		}
	}

	// Add padding
	widths.namespace += 2
	widths.name += 2
	widths.cluster += 2
	widths.ready += 2
	widths.status += 2
	widths.restarts += 2

	return widths
}

// formatDeployments formats deployment resources
func (a *TableAggregator) formatDeployments(items []ItemWithCluster) error {
	// Calculate column widths dynamically
	widths := a.calculateDeploymentColumnWidths(items)

	// Header
	fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*s %-*s %s\n",
		widths.namespace, "NAMESPACE",
		widths.name, "NAME",
		widths.cluster, "CLUSTER",
		widths.ready, "READY",
		widths.upToDate, "UP-TO-DATE",
		widths.available, "AVAILABLE",
		"AGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")

		replicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "replicas")
		readyReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "readyReplicas")
		updatedReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "updatedReplicas")
		availableReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "availableReplicas")

		ready := fmt.Sprintf("%d/%d", readyReplicas, replicas)

		age := calculateAge(item.Item)

		fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*d %-*d %s\n",
			widths.namespace, ns,
			widths.name, name,
			widths.cluster, item.Cluster,
			widths.ready, ready,
			widths.upToDate, updatedReplicas,
			widths.available, availableReplicas,
			age)
	}

	return nil
}

// calculateDeploymentColumnWidths calculates optimal column widths for deployment table
func (a *TableAggregator) calculateDeploymentColumnWidths(items []ItemWithCluster) deploymentColumnWidths {
	widths := deploymentColumnWidths{
		namespace: len("NAMESPACE"),
		name:      len("NAME"),
		cluster:   len("CLUSTER"),
		ready:     len("READY"),
		upToDate:  len("UP-TO-DATE"),
		available: len("AVAILABLE"),
	}

	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		replicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "replicas")
		readyReplicas, _, _ := unstructured.NestedInt64(item.Item.Object, "status", "readyReplicas")

		if len(ns) > widths.namespace {
			widths.namespace = len(ns)
		}
		if len(name) > widths.name {
			widths.name = len(name)
		}
		if len(item.Cluster) > widths.cluster {
			widths.cluster = len(item.Cluster)
		}

		readyStr := fmt.Sprintf("%d/%d", readyReplicas, replicas)
		if len(readyStr) > widths.ready {
			widths.ready = len(readyStr)
		}
	}

	// Add padding
	widths.namespace += 2
	widths.name += 2
	widths.cluster += 2
	widths.ready += 2
	widths.upToDate += 2
	widths.available += 2

	return widths
}

// formatServices formats service resources
func (a *TableAggregator) formatServices(items []ItemWithCluster) error {
	// Calculate column widths dynamically
	widths := a.calculateServiceColumnWidths(items)

	// Header
	fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*s %-*s %-*s %s\n",
		widths.namespace, "NAMESPACE",
		widths.name, "NAME",
		widths.cluster, "CLUSTER",
		widths.svcType, "TYPE",
		widths.clusterIP, "CLUSTER-IP",
		widths.externalIP, "EXTERNAL-IP",
		widths.ports, "PORT(S)",
		"AGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")

		svcType, _, _ := unstructured.NestedString(item.Item.Object, "spec", "type")
		clusterIP, _, _ := unstructured.NestedString(item.Item.Object, "spec", "clusterIP")
		externalIP := noneValue

		// Get ports
		ports := noneValue
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

		age := calculateAge(item.Item)

		fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %-*s %-*s %-*s %s\n",
			widths.namespace, ns,
			widths.name, name,
			widths.cluster, item.Cluster,
			widths.svcType, svcType,
			widths.clusterIP, clusterIP,
			widths.externalIP, externalIP,
			widths.ports, ports,
			age)
	}

	return nil
}

// calculateServiceColumnWidths calculates optimal column widths for service table
func (a *TableAggregator) calculateServiceColumnWidths(items []ItemWithCluster) serviceColumnWidths {
	widths := serviceColumnWidths{
		namespace:  len("NAMESPACE"),
		name:       len("NAME"),
		cluster:    len("CLUSTER"),
		svcType:    len("TYPE"),
		clusterIP:  len("CLUSTER-IP"),
		externalIP: len("EXTERNAL-IP"),
		ports:      len("PORT(S)"),
	}

	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		svcType, _, _ := unstructured.NestedString(item.Item.Object, "spec", "type")
		clusterIP, _, _ := unstructured.NestedString(item.Item.Object, "spec", "clusterIP")

		if len(ns) > widths.namespace {
			widths.namespace = len(ns)
		}
		if len(name) > widths.name {
			widths.name = len(name)
		}
		if len(item.Cluster) > widths.cluster {
			widths.cluster = len(item.Cluster)
		}
		if len(svcType) > widths.svcType {
			widths.svcType = len(svcType)
		}
		if len(clusterIP) > widths.clusterIP {
			widths.clusterIP = len(clusterIP)
		}

		// Calculate ports width
		ports := noneValue
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
		if len(ports) > widths.ports {
			widths.ports = len(ports)
		}
	}

	// Add padding
	widths.namespace += 2
	widths.name += 2
	widths.cluster += 2
	widths.svcType += 2
	widths.clusterIP += 2
	widths.externalIP += 2
	widths.ports += 2

	return widths
}

// formatGeneric formats any resource type in a generic way
func (a *TableAggregator) formatGeneric(items []ItemWithCluster) error {
	// Calculate column widths dynamically
	widths := a.calculateGenericColumnWidths(items)

	// Header
	fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %s\n",
		widths.namespace, "NAMESPACE",
		widths.name, "NAME",
		widths.cluster, "CLUSTER",
		widths.kind, "KIND",
		"AGE")

	// Rows
	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		kind := item.Item.GetKind()

		if ns == "" {
			ns = noneValue
		}

		age := calculateAge(item.Item)

		fmt.Fprintf(a.writer, "%-*s %-*s %-*s %-*s %s\n",
			widths.namespace, ns,
			widths.name, name,
			widths.cluster, item.Cluster,
			widths.kind, kind,
			age)
	}

	return nil
}

// calculateGenericColumnWidths calculates optimal column widths for generic table
func (a *TableAggregator) calculateGenericColumnWidths(items []ItemWithCluster) genericColumnWidths {
	widths := genericColumnWidths{
		namespace: len("NAMESPACE"),
		name:      len("NAME"),
		cluster:   len("CLUSTER"),
		kind:      len("KIND"),
	}

	for _, item := range items {
		ns, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Item.Object, "metadata", "name")
		kind := item.Item.GetKind()

		if ns == "" {
			ns = "<none>"
		}

		if len(ns) > widths.namespace {
			widths.namespace = len(ns)
		}
		if len(name) > widths.name {
			widths.name = len(name)
		}
		if len(item.Cluster) > widths.cluster {
			widths.cluster = len(item.Cluster)
		}
		if len(kind) > widths.kind {
			widths.kind = len(kind)
		}
	}

	// Add padding
	widths.namespace += 2
	widths.name += 2
	widths.cluster += 2
	widths.kind += 2

	return widths
}

// calculateAge calculates the age of a resource from its creation timestamp
func calculateAge(obj unstructured.Unstructured) string {
	creationTime, found, _ := unstructured.NestedString(obj.Object, "metadata", "creationTimestamp")
	if !found || creationTime == "" {
		return noneValue
	}

	// Parse the timestamp
	t, err := time.Parse(time.RFC3339, creationTime)
	if err != nil {
		return noneValue
	}

	// Calculate duration
	duration := time.Since(t)

	// Format like kubectl does
	return formatDuration(duration)
}

// formatDuration formats a duration in kubectl style (e.g., "5m", "2h", "3d")
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}
