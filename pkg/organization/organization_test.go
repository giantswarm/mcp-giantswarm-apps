package organization

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsOrganizationNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      bool
	}{
		{
			name:      "valid organization namespace",
			namespace: "org-giantswarm",
			want:      true,
		},
		{
			name:      "valid organization namespace with hyphen",
			namespace: "org-my-company",
			want:      true,
		},
		{
			name:      "not an organization namespace",
			namespace: "giantswarm",
			want:      false,
		},
		{
			name:      "workload cluster namespace",
			namespace: "workload-cluster1",
			want:      false,
		},
		{
			name:      "system namespace",
			namespace: "kube-system",
			want:      false,
		},
		{
			name:      "empty namespace",
			namespace: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOrganizationNamespace(tt.namespace); got != tt.want {
				t.Errorf("IsOrganizationNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkloadClusterNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      bool
	}{
		{
			name:      "valid workload cluster namespace",
			namespace: "workload-cluster1",
			want:      true,
		},
		{
			name:      "valid workload cluster namespace with hyphen",
			namespace: "workload-prod-cluster",
			want:      true,
		},
		{
			name:      "organization namespace",
			namespace: "org-giantswarm",
			want:      false,
		},
		{
			name:      "system namespace",
			namespace: "kube-system",
			want:      false,
		},
		{
			name:      "empty namespace",
			namespace: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkloadClusterNamespace(tt.namespace); got != tt.want {
				t.Errorf("IsWorkloadClusterNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOrganizationFromNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      string
		wantErr   bool
	}{
		{
			name:      "valid organization namespace",
			namespace: "org-giantswarm",
			want:      "giantswarm",
			wantErr:   false,
		},
		{
			name:      "valid organization namespace with hyphen",
			namespace: "org-my-company",
			want:      "my-company",
			wantErr:   false,
		},
		{
			name:      "not an organization namespace",
			namespace: "giantswarm",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "workload cluster namespace",
			namespace: "workload-cluster1",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOrganizationFromNamespace(tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrganizationFromNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetOrganizationFromNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOrganizationNamespace(t *testing.T) {
	tests := []struct {
		name         string
		organization string
		want         string
	}{
		{
			name:         "simple organization",
			organization: "giantswarm",
			want:         "org-giantswarm",
		},
		{
			name:         "organization with hyphen",
			organization: "my-company",
			want:         "org-my-company",
		},
		{
			name:         "already has prefix",
			organization: "org-giantswarm",
			want:         "org-giantswarm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOrganizationNamespace(tt.organization); got != tt.want {
				t.Errorf("GetOrganizationNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListOrganizationNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []runtime.Object
		want       []string
		wantErr    bool
	}{
		{
			name: "namespaces with organization label",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-giantswarm",
						Labels: map[string]string{
							OrganizationLabel: "true",
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-adidas",
						Labels: map[string]string{
							OrganizationLabel: "true",
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-system",
					},
				},
			},
			want:    []string{"org-giantswarm", "org-adidas"},
			wantErr: false,
		},
		{
			name: "namespaces without labels (fallback to prefix)",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-giantswarm",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-company",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload-cluster1",
					},
				},
			},
			want:    []string{"org-giantswarm", "org-company"},
			wantErr: false,
		},
		{
			name: "no organization namespaces",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
			want:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tt.namespaces...)
			ctx := context.Background()

			got, err := ListOrganizationNamespaces(ctx, client)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListOrganizationNamespaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ListOrganizationNamespaces() returned %d namespaces, want %d", len(got), len(tt.want))
				return
			}

			// Check if all expected namespaces are present
			for _, want := range tt.want {
				found := false
				for _, g := range got {
					if g == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ListOrganizationNamespaces() missing namespace %s", want)
				}
			}
		})
	}
}

func TestGetNamespacesByOrganization(t *testing.T) {
	tests := []struct {
		name         string
		organization string
		namespaces   []runtime.Object
		want         []string
		wantErr      bool
	}{
		{
			name:         "organization with multiple namespaces",
			organization: "giantswarm",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "org-giantswarm",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload-cluster1",
						Labels: map[string]string{
							"giantswarm.io/owner": "giantswarm",
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workload-cluster2",
						Labels: map[string]string{
							"giantswarm.io/owner": "adidas",
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-namespace",
						Labels: map[string]string{
							OrganizationLabel: "giantswarm",
						},
					},
				},
			},
			want:    []string{"org-giantswarm", "workload-cluster1", "some-namespace"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tt.namespaces...)
			ctx := context.Background()

			got, err := GetNamespacesByOrganization(ctx, client, tt.organization)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNamespacesByOrganization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetNamespacesByOrganization() returned %d namespaces, want %d", len(got), len(tt.want))
				return
			}

			// Check if all expected namespaces are present
			for _, want := range tt.want {
				found := false
				for _, g := range got {
					if g == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetNamespacesByOrganization() missing namespace %s", want)
				}
			}
		})
	}
}

func TestGetNamespaceInfo(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		ns        *corev1.Namespace
		want      *NamespaceInfo
		wantErr   bool
	}{
		{
			name:      "organization namespace",
			namespace: "org-giantswarm",
			ns: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "org-giantswarm",
					Labels: map[string]string{
						OrganizationLabel: "true",
					},
				},
			},
			want: &NamespaceInfo{
				Name:         "org-giantswarm",
				Type:         NamespaceTypeOrganization,
				Organization: "giantswarm",
				Labels: map[string]string{
					OrganizationLabel: "true",
				},
			},
			wantErr: false,
		},
		{
			name:      "workload cluster namespace",
			namespace: "workload-cluster1",
			ns: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "workload-cluster1",
					Labels: map[string]string{
						"giantswarm.io/cluster": "cluster1",
						"giantswarm.io/owner":   "giantswarm",
					},
				},
			},
			want: &NamespaceInfo{
				Name:         "workload-cluster1",
				Type:         NamespaceTypeWorkloadCluster,
				Organization: "giantswarm",
				ClusterID:    "cluster1",
				Labels: map[string]string{
					"giantswarm.io/cluster": "cluster1",
					"giantswarm.io/owner":   "giantswarm",
				},
			},
			wantErr: false,
		},
		{
			name:      "system namespace",
			namespace: "kube-system",
			ns: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
			},
			want: &NamespaceInfo{
				Name:   "kube-system",
				Type:   NamespaceTypeSystem,
				Labels: map[string]string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tt.ns)
			ctx := context.Background()

			got, err := GetNamespaceInfo(ctx, client, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNamespaceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("GetNamespaceInfo() Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Type != tt.want.Type {
				t.Errorf("GetNamespaceInfo() Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Organization != tt.want.Organization {
				t.Errorf("GetNamespaceInfo() Organization = %v, want %v", got.Organization, tt.want.Organization)
			}
			if got.ClusterID != tt.want.ClusterID {
				t.Errorf("GetNamespaceInfo() ClusterID = %v, want %v", got.ClusterID, tt.want.ClusterID)
			}
		})
	}
}
