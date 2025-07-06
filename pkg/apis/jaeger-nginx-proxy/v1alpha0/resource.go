package v1alpha0

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JaegerNginxProxyStatus struct {
	// Add your custom status fields here
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
	// You can add more fields as needed
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type JaegerNginxProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JaegerNginxProxySpec   `json:"spec"`
	Status JaegerNginxProxyStatus `json:"status,omitempty"`
}

// JaegerNginxProxySpec defines the desired state of JaegerNginxProxy
type JaegerNginxProxySpec struct {
	ReplicaCount  int       `json:"replicaCount"`
	Upstream      Upstream  `json:"upstream"`
	ContainerPort int       `json:"containerPort"`
	Image         Image     `json:"image"`
	Ports         []Port    `json:"ports"`
	Service       Service   `json:"service"`
	Resources     Resources `json:"resources"`
}

type Upstream struct {
	CollectorHost string `json:"collectorHost"`
}

type Port struct {
	Name string `json:"name"`
	Port int    `json:"port"`
	Path string `json:"path"`
}

type Service struct {
	Type string `json:"type"`
}

type Resources struct {
	Limits   Resource `json:"limits"`
	Requests Resource `json:"requests"`
}

type Resource struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	PullPolicy string `json:"pullPolicy"`
}

// +kubebuilder:object:root=true
type JaegerNginxProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []JaegerNginxProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JaegerNginxProxy{}, &JaegerNginxProxyList{})
}
