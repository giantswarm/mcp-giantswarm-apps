package app

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// App represents a Giant Swarm App resource
type App struct {
	Name      string
	Namespace string
	Spec      AppSpec
	Status    AppStatus
}

// AppSpec represents the spec of an App
type AppSpec struct {
	Catalog    string
	Name       string
	Namespace  string
	Version    string
	KubeConfig KubeConfig
	Config     *AppConfig
	UserConfig *AppConfig
}

// KubeConfig represents the kubeconfig for the app
type KubeConfig struct {
	InCluster bool
	Secret    *SecretReference
}

// SecretReference references a secret
type SecretReference struct {
	Name      string
	Namespace string
}

// AppConfig represents app configuration
type AppConfig struct {
	ConfigMap *ConfigMapReference
	Secret    *SecretReference
}

// ConfigMapReference references a configmap
type ConfigMapReference struct {
	Name      string
	Namespace string
}

// AppStatus represents the status of an App
type AppStatus struct {
	AppVersion string
	Version    string
	Release    ReleaseStatus
}

// ReleaseStatus represents the Helm release status
type ReleaseStatus struct {
	LastDeployed string
	Status       string
}

// NewAppFromUnstructured converts an unstructured object to an App
func NewAppFromUnstructured(obj *unstructured.Unstructured) (*App, error) {
	app := &App{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return app, err
	}

	// Catalog
	if catalog, ok := spec["catalog"].(string); ok {
		app.Spec.Catalog = catalog
	}

	// Name
	if name, ok := spec["name"].(string); ok {
		app.Spec.Name = name
	}

	// Namespace
	if namespace, ok := spec["namespace"].(string); ok {
		app.Spec.Namespace = namespace
	}

	// Version
	if version, ok := spec["version"].(string); ok {
		app.Spec.Version = version
	}

	// KubeConfig
	if kubeConfig, ok := spec["kubeConfig"].(map[string]interface{}); ok {
		if inCluster, ok := kubeConfig["inCluster"].(bool); ok {
			app.Spec.KubeConfig.InCluster = inCluster
		}
	}

	// Config
	if config, ok := spec["config"].(map[string]interface{}); ok {
		app.Spec.Config = parseAppConfig(config)
	}

	// UserConfig
	if userConfig, ok := spec["userConfig"].(map[string]interface{}); ok {
		app.Spec.UserConfig = parseAppConfig(userConfig)
	}

	// Extract status
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err == nil && found {
		// AppVersion
		if appVersion, ok := status["appVersion"].(string); ok {
			app.Status.AppVersion = appVersion
		}

		// Version
		if version, ok := status["version"].(string); ok {
			app.Status.Version = version
		}

		// Release
		if release, ok := status["release"].(map[string]interface{}); ok {
			if lastDeployed, ok := release["lastDeployed"].(string); ok {
				app.Status.Release.LastDeployed = lastDeployed
			}
			if releaseStatus, ok := release["status"].(string); ok {
				app.Status.Release.Status = releaseStatus
			}
		}
	}

	return app, nil
}

// parseAppConfig parses app config from unstructured data
func parseAppConfig(config map[string]interface{}) *AppConfig {
	ac := &AppConfig{}

	if configMap, ok := config["configMap"].(map[string]interface{}); ok {
		ac.ConfigMap = &ConfigMapReference{}
		if name, ok := configMap["name"].(string); ok {
			ac.ConfigMap.Name = name
		}
		if namespace, ok := configMap["namespace"].(string); ok {
			ac.ConfigMap.Namespace = namespace
		}
	}

	if secret, ok := config["secret"].(map[string]interface{}); ok {
		ac.Secret = &SecretReference{}
		if name, ok := secret["name"].(string); ok {
			ac.Secret.Name = name
		}
		if namespace, ok := secret["namespace"].(string); ok {
			ac.Secret.Namespace = namespace
		}
	}

	return ac
}

// ToUnstructured converts an App to an unstructured object
func (a *App) ToUnstructured() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "application.giantswarm.io/v1alpha1",
			"kind":       "App",
			"metadata": map[string]interface{}{
				"name":      a.Name,
				"namespace": a.Namespace,
			},
			"spec": map[string]interface{}{
				"catalog":   a.Spec.Catalog,
				"name":      a.Spec.Name,
				"namespace": a.Spec.Namespace,
				"version":   a.Spec.Version,
				"kubeConfig": map[string]interface{}{
					"inCluster": a.Spec.KubeConfig.InCluster,
				},
			},
		},
	}

	// Add config if present
	if a.Spec.Config != nil {
		spec := obj.Object["spec"].(map[string]interface{})
		config := make(map[string]interface{})

		if a.Spec.Config.ConfigMap != nil {
			config["configMap"] = map[string]interface{}{
				"name":      a.Spec.Config.ConfigMap.Name,
				"namespace": a.Spec.Config.ConfigMap.Namespace,
			}
		}

		if a.Spec.Config.Secret != nil {
			config["secret"] = map[string]interface{}{
				"name":      a.Spec.Config.Secret.Name,
				"namespace": a.Spec.Config.Secret.Namespace,
			}
		}

		spec["config"] = config
	}

	// Add userConfig if present
	if a.Spec.UserConfig != nil {
		spec := obj.Object["spec"].(map[string]interface{})
		userConfig := make(map[string]interface{})

		if a.Spec.UserConfig.ConfigMap != nil {
			userConfig["configMap"] = map[string]interface{}{
				"name":      a.Spec.UserConfig.ConfigMap.Name,
				"namespace": a.Spec.UserConfig.ConfigMap.Namespace,
			}
		}

		if a.Spec.UserConfig.Secret != nil {
			userConfig["secret"] = map[string]interface{}{
				"name":      a.Spec.UserConfig.Secret.Name,
				"namespace": a.Spec.UserConfig.Secret.Namespace,
			}
		}

		spec["userConfig"] = userConfig
	}

	return obj
}
