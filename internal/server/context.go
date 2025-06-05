package server

import (
	"github.com/giantswarm/mcp-giantswarm-apps/internal/k8s"
)

// Context holds shared server resources
type Context struct {
	K8sClient     *k8s.Client
	DynamicClient *k8s.DynamicClient
}

// NewContext creates a new server context
func NewContext(k8sClient *k8s.Client, dynamicClient *k8s.DynamicClient) *Context {
	return &Context{
		K8sClient:     k8sClient,
		DynamicClient: dynamicClient,
	}
} 