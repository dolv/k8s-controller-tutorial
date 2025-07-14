package informer

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	deploymentInformer cache.SharedIndexInformer
	allowedNamespaces  map[string]bool
)

// StartDeploymentInformer starts a shared informer for Deployments in the specified namespaces.
// serverNamespace can be:
// - "all" or empty: watch all namespaces
// - "default": watch only default namespace
// - "ns1,ns2,ns3": watch specific comma-separated namespaces
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, serverNamespace string) {
	var factory informers.SharedInformerFactory

	if serverNamespace == "" || serverNamespace == "all" {
		// Watch all namespaces
		factory = informers.NewSharedInformerFactoryWithOptions(
			clientset,
			30*time.Second,
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fields.Everything().String()
			}),
		)
		log.Info().Msg("Starting deployment informer for ALL namespaces")
	} else if strings.Contains(serverNamespace, ",") {
		// Watch multiple specific namespaces
		namespaceList := strings.Split(serverNamespace, ",")
		// Trim whitespace from each namespace
		for i, ns := range namespaceList {
			namespaceList[i] = strings.TrimSpace(ns)
		}
		log.Info().Msgf("Starting deployment informer for namespaces: %v", namespaceList)

		// For multiple namespaces, we need to create separate informers
		// This is a limitation of the informer factory - it can only watch one namespace at a time
		// So we'll create a factory without namespace restriction and filter in our handlers
		factory = informers.NewSharedInformerFactoryWithOptions(
			clientset,
			30*time.Second,
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fields.Everything().String()
			}),
		)
		// Store the allowed namespaces for filtering
		setAllowedNamespaces(namespaceList)
	} else {
		// Watch specific single namespace
		factory = informers.NewSharedInformerFactoryWithOptions(
			clientset,
			30*time.Second,
			informers.WithNamespace(serverNamespace),
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fields.Everything().String()
			}),
		)
		log.Info().Msgf("Starting deployment informer for namespace: %s", serverNamespace)
	}
	log.Debug().Msg("Creating Informer instance")
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			name := getDeploymentName(obj)
			log.Info().Msgf("[INFORMER][Add] Deployment added: %s", name)
			log.Debug().Msgf("[INFORMER][Add] Cache now contains %d deployments", len(GetDeploymentNames()))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDep, oldOk := oldObj.(*appsv1.Deployment)
			newDep, newOk := newObj.(*appsv1.Deployment)

			if !oldOk || !newOk {
				log.Info().Msgf("[INFORMER][Update] Deployment updated: %s (type conversion failed)", getDeploymentName(newObj))
				return
			}

			name := getDeploymentName(newObj)
			changes := []string{}

			// Compare replicas
			if oldDep.Spec.Replicas != nil && newDep.Spec.Replicas != nil {
				if *oldDep.Spec.Replicas != *newDep.Spec.Replicas {
					changes = append(changes, fmt.Sprintf("replicas: %d -> %d", *oldDep.Spec.Replicas, *newDep.Spec.Replicas))
				}
			}

			// Compare image
			if len(oldDep.Spec.Template.Spec.Containers) > 0 && len(newDep.Spec.Template.Spec.Containers) > 0 {
				if oldDep.Spec.Template.Spec.Containers[0].Image != newDep.Spec.Template.Spec.Containers[0].Image {
					changes = append(changes, fmt.Sprintf("image: %s -> %s",
						oldDep.Spec.Template.Spec.Containers[0].Image,
						newDep.Spec.Template.Spec.Containers[0].Image))
				}
			}

			// Compare labels
			if !reflect.DeepEqual(oldDep.Labels, newDep.Labels) {
				changes = append(changes, "labels changed")
			}

			// Compare annotations
			if !reflect.DeepEqual(oldDep.Annotations, newDep.Annotations) {
				changes = append(changes, "annotations changed")
			}

			// Check if it's just a status update
			if len(changes) == 0 {
				statusChanges := []string{}
				if oldDep.Status.Replicas != newDep.Status.Replicas {
					statusChanges = append(statusChanges,
						fmt.Sprintf("status.replicas: %d -> %d", oldDep.Status.Replicas, newDep.Status.Replicas))
				}
				if oldDep.Status.AvailableReplicas != newDep.Status.AvailableReplicas {
					statusChanges = append(statusChanges,
						fmt.Sprintf("status.availableReplicas: %d -> %d", oldDep.Status.AvailableReplicas, newDep.Status.AvailableReplicas))
				}
				if oldDep.Status.UpdatedReplicas != newDep.Status.UpdatedReplicas {
					statusChanges = append(statusChanges,
						fmt.Sprintf("status.updatedReplicas: %d -> %d", oldDep.Status.UpdatedReplicas, newDep.Status.UpdatedReplicas))
				}
				if oldDep.Status.ReadyReplicas != newDep.Status.ReadyReplicas {
					statusChanges = append(statusChanges,
						fmt.Sprintf("status.readyReplicas: %d -> %d", oldDep.Status.ReadyReplicas, newDep.Status.ReadyReplicas))
				}
				if oldDep.Status.UnavailableReplicas != newDep.Status.UnavailableReplicas {
					statusChanges = append(statusChanges,
						fmt.Sprintf("status.unavailableReplicas: %d -> %d", oldDep.Status.UnavailableReplicas, newDep.Status.UnavailableReplicas))
				}
				// Add more status fields as needed
				if len(statusChanges) > 0 {
					log.Info().Msgf("[INFORMER][Update] Deployment status updated: %s - Changes: %s",
						name, strings.Join(statusChanges, ", "))
				} else {
					log.Info().Msgf("[INFORMER][Update] Deployment status updated: %s (generation: %d -> %d)",
						name, oldDep.Generation, newDep.Generation)
				}
			} else {
				log.Info().Msgf("[INFORMER][Update] Deployment updated: %s - Changes: %s",
					name, strings.Join(changes, ", "))
			}
		},
		DeleteFunc: func(obj interface{}) {
			name := getDeploymentName(obj)
			log.Info().Msgf("[INFORMER][Delete] Deployment deleted: %s", name)
			log.Debug().Msgf("[INFORMER][Delete] Cache now contains %d deployments", len(GetDeploymentNames()))
		},
	})

	log.Info().Msg("Starting deployment informer...")
	factory.Start(ctx.Done())
	for t, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			log.Error().Msgf("Failed to sync informer for %v", t)
			os.Exit(1)
		}
	}
	log.Info().Msg("Deployment informer cache synced. Watching for events...")
	<-ctx.Done() // Block until context is cancelled
}

// GetDeploymentNames returns a slice of deployment names from the informer's cache.
func GetDeploymentNames() []string {
	var names []string
	if deploymentInformer == nil {
		log.Warn().Msg("Deployment informer is nil, returning empty list")
		return names
	}
	for _, obj := range deploymentInformer.GetStore().List() {
		if d, ok := obj.(*appsv1.Deployment); ok {
			if isNamespaceAllowed(d.Namespace) {
				names = append(names, d.Name)
			}
		}
	}
	log.Debug().Msgf("Found %d deployments in cache", len(names))
	return names
}

// setAllowedNamespaces sets the list of namespaces that should be included in results
func setAllowedNamespaces(namespaces []string) {
	allowedNamespaces = make(map[string]bool)
	for _, ns := range namespaces {
		allowedNamespaces[ns] = true
	}
}

// isNamespaceAllowed checks if a namespace is in the allowed list
func isNamespaceAllowed(namespace string) bool {
	if allowedNamespaces == nil {
		return true // If no restrictions, allow all
	}
	return allowedNamespaces[namespace]
}

// GetDeploymentNamesWithNamespace returns a slice of deployment names with their namespaces from the informer's cache.
func GetDeploymentNamesWithNamespace() []map[string]string {
	var deployments []map[string]string
	if deploymentInformer == nil {
		log.Warn().Msg("Deployment informer is nil, returning empty list")
		return deployments
	}
	for _, obj := range deploymentInformer.GetStore().List() {
		if d, ok := obj.(*appsv1.Deployment); ok {
			if isNamespaceAllowed(d.Namespace) {
				deployments = append(deployments, map[string]string{
					"name":      d.Name,
					"namespace": d.Namespace,
				})
			}
		}
	}
	log.Debug().Msgf("Found %d deployments in cache", len(deployments))
	return deployments
}

func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
