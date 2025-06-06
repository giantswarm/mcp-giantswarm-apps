package cluster

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ClusterGVK is the GroupVersionKind for CAPI Cluster resources
var ClusterGVK = schema.GroupVersionKind{
	Group:   "cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "Cluster",
}

// Cluster represents a CAPI Cluster resource
type Cluster struct {
	Name      string
	Namespace string
	Spec      ClusterSpec
	Status    ClusterStatus
	Labels    map[string]string
}

// ClusterSpec represents the spec of a CAPI Cluster
type ClusterSpec struct {
	ClusterNetwork    *ClusterNetwork
	InfrastructureRef *ObjectReference
	ControlPlaneRef   *ObjectReference
}

// ClusterNetwork represents cluster networking configuration
type ClusterNetwork struct {
	APIServerPort *int32
	Services      *NetworkRanges
	Pods          *NetworkRanges
}

// NetworkRanges represents CIDR ranges for network configuration
type NetworkRanges struct {
	CIDRBlocks []string
}

// ObjectReference references another Kubernetes object
type ObjectReference struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
}

// ClusterStatus represents the status of a CAPI Cluster
type ClusterStatus struct {
	Phase               string
	InfrastructureReady bool
	ControlPlaneReady   bool
	Conditions          []Condition
}

// Condition represents a condition of a cluster
type Condition struct {
	Type               string
	Status             string
	LastTransitionTime string
	Reason             string
	Message            string
}

// IsReady returns true if the cluster is ready
func (c *Cluster) IsReady() bool {
	return c.Status.Phase == "Provisioned" &&
		c.Status.InfrastructureReady &&
		c.Status.ControlPlaneReady
}

// GetOrganization returns the organization that owns this cluster
func (c *Cluster) GetOrganization() string {
	if org, ok := c.Labels["giantswarm.io/organization"]; ok {
		return org
	}
	return ""
}

// GetProvider returns the infrastructure provider of this cluster
func (c *Cluster) GetProvider() string {
	if provider, ok := c.Labels["cluster.x-k8s.io/provider"]; ok {
		return provider
	}
	if c.Spec.InfrastructureRef != nil {
		// Extract provider from infrastructure kind (e.g., AWSCluster -> aws)
		kind := c.Spec.InfrastructureRef.Kind
		if len(kind) > 7 && kind[len(kind)-7:] == "Cluster" {
			return kind[:len(kind)-7]
		}
	}
	return "unknown"
}

// NewClusterFromUnstructured converts an unstructured object to a Cluster
func NewClusterFromUnstructured(obj *unstructured.Unstructured) (*Cluster, error) {
	cluster := &Cluster{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Labels:    obj.GetLabels(),
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err == nil && found {
		// ClusterNetwork
		if clusterNetwork, ok := spec["clusterNetwork"].(map[string]interface{}); ok {
			cluster.Spec.ClusterNetwork = parseClusterNetwork(clusterNetwork)
		}

		// InfrastructureRef
		if infraRef, ok := spec["infrastructureRef"].(map[string]interface{}); ok {
			cluster.Spec.InfrastructureRef = parseObjectReference(infraRef)
		}

		// ControlPlaneRef
		if cpRef, ok := spec["controlPlaneRef"].(map[string]interface{}); ok {
			cluster.Spec.ControlPlaneRef = parseObjectReference(cpRef)
		}
	}

	// Extract status
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err == nil && found {
		// Phase
		if phase, ok := status["phase"].(string); ok {
			cluster.Status.Phase = phase
		}

		// InfrastructureReady
		if ready, ok := status["infrastructureReady"].(bool); ok {
			cluster.Status.InfrastructureReady = ready
		}

		// ControlPlaneReady
		if ready, ok := status["controlPlaneReady"].(bool); ok {
			cluster.Status.ControlPlaneReady = ready
		}

		// Conditions
		if conditions, ok := status["conditions"].([]interface{}); ok {
			cluster.Status.Conditions = parseConditions(conditions)
		}
	}

	return cluster, nil
}

// ToUnstructured converts a Cluster to an unstructured object
func (c *Cluster) ToUnstructured() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(ClusterGVK)
	obj.SetName(c.Name)
	obj.SetNamespace(c.Namespace)
	obj.SetLabels(c.Labels)

	spec := make(map[string]interface{})

	if c.Spec.ClusterNetwork != nil {
		spec["clusterNetwork"] = clusterNetworkToMap(c.Spec.ClusterNetwork)
	}

	if c.Spec.InfrastructureRef != nil {
		spec["infrastructureRef"] = objectReferenceToMap(c.Spec.InfrastructureRef)
	}

	if c.Spec.ControlPlaneRef != nil {
		spec["controlPlaneRef"] = objectReferenceToMap(c.Spec.ControlPlaneRef)
	}

	obj.Object["spec"] = spec

	return obj
}

// Helper functions

func parseClusterNetwork(data map[string]interface{}) *ClusterNetwork {
	cn := &ClusterNetwork{}

	if port, ok := data["apiServerPort"].(int64); ok {
		p := int32(port)
		cn.APIServerPort = &p
	}

	if services, ok := data["services"].(map[string]interface{}); ok {
		cn.Services = parseNetworkRanges(services)
	}

	if pods, ok := data["pods"].(map[string]interface{}); ok {
		cn.Pods = parseNetworkRanges(pods)
	}

	return cn
}

func parseNetworkRanges(data map[string]interface{}) *NetworkRanges {
	nr := &NetworkRanges{}

	if cidrBlocks, ok := data["cidrBlocks"].([]interface{}); ok {
		for _, cidr := range cidrBlocks {
			if cidrStr, ok := cidr.(string); ok {
				nr.CIDRBlocks = append(nr.CIDRBlocks, cidrStr)
			}
		}
	}

	return nr
}

func parseObjectReference(data map[string]interface{}) *ObjectReference {
	ref := &ObjectReference{}

	if apiVersion, ok := data["apiVersion"].(string); ok {
		ref.APIVersion = apiVersion
	}
	if kind, ok := data["kind"].(string); ok {
		ref.Kind = kind
	}
	if name, ok := data["name"].(string); ok {
		ref.Name = name
	}
	if namespace, ok := data["namespace"].(string); ok {
		ref.Namespace = namespace
	}

	return ref
}

func parseConditions(data []interface{}) []Condition {
	conditions := make([]Condition, 0)

	for _, c := range data {
		if cond, ok := c.(map[string]interface{}); ok {
			condition := Condition{}

			if t, ok := cond["type"].(string); ok {
				condition.Type = t
			}
			if s, ok := cond["status"].(string); ok {
				condition.Status = s
			}
			if time, ok := cond["lastTransitionTime"].(string); ok {
				condition.LastTransitionTime = time
			}
			if r, ok := cond["reason"].(string); ok {
				condition.Reason = r
			}
			if m, ok := cond["message"].(string); ok {
				condition.Message = m
			}

			conditions = append(conditions, condition)
		}
	}

	return conditions
}

func clusterNetworkToMap(cn *ClusterNetwork) map[string]interface{} {
	result := make(map[string]interface{})

	if cn.APIServerPort != nil {
		result["apiServerPort"] = *cn.APIServerPort
	}

	if cn.Services != nil {
		result["services"] = networkRangesToMap(cn.Services)
	}

	if cn.Pods != nil {
		result["pods"] = networkRangesToMap(cn.Pods)
	}

	return result
}

func networkRangesToMap(nr *NetworkRanges) map[string]interface{} {
	return map[string]interface{}{
		"cidrBlocks": nr.CIDRBlocks,
	}
}

func objectReferenceToMap(ref *ObjectReference) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": ref.APIVersion,
		"kind":       ref.Kind,
		"name":       ref.Name,
		"namespace":  ref.Namespace,
	}
}
