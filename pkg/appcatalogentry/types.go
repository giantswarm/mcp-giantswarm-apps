package appcatalogentry

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AppCatalogEntry represents a Giant Swarm AppCatalogEntry resource
type AppCatalogEntry struct {
	Name        string
	Namespace   string
	Spec        AppCatalogEntrySpec
	Labels      map[string]string
	Annotations map[string]string
}

// AppCatalogEntrySpec represents the spec of an AppCatalogEntry
type AppCatalogEntrySpec struct {
	AppName     string
	AppVersion  string
	Catalog     CatalogReference
	Chart       ChartSpec
	DateCreated *time.Time
	DateUpdated *time.Time
	Restrictions *Restrictions
}

// CatalogReference references the catalog this entry belongs to
type CatalogReference struct {
	Name      string
	Namespace string
}

// ChartSpec contains chart metadata
type ChartSpec struct {
	APIVersion  string
	AppVersion  string
	Description string
	Home        string
	Icon        string
	Keywords    []string
	Name        string
	Sources     []string
	URLs        []string
	Version     string
}

// Restrictions defines restrictions for the app
type Restrictions struct {
	ClusterSingleton  bool
	NamespaceSingleton bool
	FixedNamespace    string
	GpuInstances      bool
}

// GetLatestVersion returns the latest version from the entry
func (e *AppCatalogEntry) GetLatestVersion() string {
	if e.Spec.Chart.Version != "" {
		return e.Spec.Chart.Version
	}
	return e.Spec.AppVersion
}

// GetAppVersion returns the application version
func (e *AppCatalogEntry) GetAppVersion() string {
	if e.Spec.Chart.AppVersion != "" {
		return e.Spec.Chart.AppVersion
	}
	return e.Spec.AppVersion
}

// IsClusterApp returns true if this is a cluster-wide app
func (e *AppCatalogEntry) IsClusterApp() bool {
	if e.Spec.Restrictions != nil {
		return e.Spec.Restrictions.ClusterSingleton
	}
	return false
}

// NewAppCatalogEntryFromUnstructured converts an unstructured object to an AppCatalogEntry
func NewAppCatalogEntryFromUnstructured(obj *unstructured.Unstructured) (*AppCatalogEntry, error) {
	entry := &AppCatalogEntry{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return entry, err
	}

	// AppName
	if appName, ok := spec["appName"].(string); ok {
		entry.Spec.AppName = appName
	}

	// AppVersion
	if appVersion, ok := spec["appVersion"].(string); ok {
		entry.Spec.AppVersion = appVersion
	}

	// Catalog reference
	if catalog, ok := spec["catalog"].(map[string]interface{}); ok {
		if name, ok := catalog["name"].(string); ok {
			entry.Spec.Catalog.Name = name
		}
		if namespace, ok := catalog["namespace"].(string); ok {
			entry.Spec.Catalog.Namespace = namespace
		}
	}

	// Chart spec
	if chart, ok := spec["chart"].(map[string]interface{}); ok {
		entry.Spec.Chart = parseChartSpec(chart)
	}

	// Dates
	if dateCreated, ok := spec["dateCreated"].(string); ok {
		if t, err := time.Parse(time.RFC3339, dateCreated); err == nil {
			entry.Spec.DateCreated = &t
		}
	}
	if dateUpdated, ok := spec["dateUpdated"].(string); ok {
		if t, err := time.Parse(time.RFC3339, dateUpdated); err == nil {
			entry.Spec.DateUpdated = &t
		}
	}

	// Restrictions
	if restrictions, ok := spec["restrictions"].(map[string]interface{}); ok {
		entry.Spec.Restrictions = parseRestrictions(restrictions)
	}

	return entry, nil
}

// parseChartSpec parses chart spec from unstructured data
func parseChartSpec(chart map[string]interface{}) ChartSpec {
	cs := ChartSpec{}

	if apiVersion, ok := chart["apiVersion"].(string); ok {
		cs.APIVersion = apiVersion
	}
	if appVersion, ok := chart["appVersion"].(string); ok {
		cs.AppVersion = appVersion
	}
	if description, ok := chart["description"].(string); ok {
		cs.Description = description
	}
	if home, ok := chart["home"].(string); ok {
		cs.Home = home
	}
	if icon, ok := chart["icon"].(string); ok {
		cs.Icon = icon
	}
	if name, ok := chart["name"].(string); ok {
		cs.Name = name
	}
	if version, ok := chart["version"].(string); ok {
		cs.Version = version
	}

	// Keywords
	if keywords, ok := chart["keywords"].([]interface{}); ok {
		cs.Keywords = make([]string, 0, len(keywords))
		for _, kw := range keywords {
			if keyword, ok := kw.(string); ok {
				cs.Keywords = append(cs.Keywords, keyword)
			}
		}
	}

	// Sources
	if sources, ok := chart["sources"].([]interface{}); ok {
		cs.Sources = make([]string, 0, len(sources))
		for _, src := range sources {
			if source, ok := src.(string); ok {
				cs.Sources = append(cs.Sources, source)
			}
		}
	}

	// URLs
	if urls, ok := chart["urls"].([]interface{}); ok {
		cs.URLs = make([]string, 0, len(urls))
		for _, u := range urls {
			if url, ok := u.(string); ok {
				cs.URLs = append(cs.URLs, url)
			}
		}
	}

	return cs
}

// parseRestrictions parses restrictions from unstructured data
func parseRestrictions(restrictions map[string]interface{}) *Restrictions {
	r := &Restrictions{}

	if clusterSingleton, ok := restrictions["clusterSingleton"].(bool); ok {
		r.ClusterSingleton = clusterSingleton
	}
	if namespaceSingleton, ok := restrictions["namespaceSingleton"].(bool); ok {
		r.NamespaceSingleton = namespaceSingleton
	}
	if fixedNamespace, ok := restrictions["fixedNamespace"].(string); ok {
		r.FixedNamespace = fixedNamespace
	}
	if gpuInstances, ok := restrictions["gpuInstances"].(bool); ok {
		r.GpuInstances = gpuInstances
	}

	return r
}

// ToUnstructured converts an AppCatalogEntry to an unstructured object
func (e *AppCatalogEntry) ToUnstructured() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "application.giantswarm.io/v1alpha1",
			"kind":       "AppCatalogEntry",
			"metadata": map[string]interface{}{
				"name":        e.Name,
				"namespace":   e.Namespace,
				"labels":      e.Labels,
				"annotations": e.Annotations,
			},
			"spec": map[string]interface{}{
				"appName":    e.Spec.AppName,
				"appVersion": e.Spec.AppVersion,
				"catalog": map[string]interface{}{
					"name":      e.Spec.Catalog.Name,
					"namespace": e.Spec.Catalog.Namespace,
				},
			},
		},
	}

	spec := obj.Object["spec"].(map[string]interface{})

	// Add chart spec
	chart := map[string]interface{}{
		"apiVersion":  e.Spec.Chart.APIVersion,
		"appVersion":  e.Spec.Chart.AppVersion,
		"description": e.Spec.Chart.Description,
		"home":        e.Spec.Chart.Home,
		"icon":        e.Spec.Chart.Icon,
		"name":        e.Spec.Chart.Name,
		"version":     e.Spec.Chart.Version,
	}

	if len(e.Spec.Chart.Keywords) > 0 {
		keywords := make([]interface{}, len(e.Spec.Chart.Keywords))
		for i, kw := range e.Spec.Chart.Keywords {
			keywords[i] = kw
		}
		chart["keywords"] = keywords
	}

	if len(e.Spec.Chart.Sources) > 0 {
		sources := make([]interface{}, len(e.Spec.Chart.Sources))
		for i, src := range e.Spec.Chart.Sources {
			sources[i] = src
		}
		chart["sources"] = sources
	}

	if len(e.Spec.Chart.URLs) > 0 {
		urls := make([]interface{}, len(e.Spec.Chart.URLs))
		for i, url := range e.Spec.Chart.URLs {
			urls[i] = url
		}
		chart["urls"] = urls
	}

	spec["chart"] = chart

	// Add dates
	if e.Spec.DateCreated != nil {
		spec["dateCreated"] = e.Spec.DateCreated.Format(time.RFC3339)
	}
	if e.Spec.DateUpdated != nil {
		spec["dateUpdated"] = e.Spec.DateUpdated.Format(time.RFC3339)
	}

	// Add restrictions
	if e.Spec.Restrictions != nil {
		spec["restrictions"] = map[string]interface{}{
			"clusterSingleton":   e.Spec.Restrictions.ClusterSingleton,
			"namespaceSingleton": e.Spec.Restrictions.NamespaceSingleton,
			"fixedNamespace":     e.Spec.Restrictions.FixedNamespace,
			"gpuInstances":       e.Spec.Restrictions.GpuInstances,
		}
	}

	return obj
} 