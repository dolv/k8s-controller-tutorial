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

var deploymentInformer cache.SharedIndexInformer

// StartDeploymentInformer starts a shared informer for Deployments in the <serverNamespace> namespace.
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, serverNamespace string) {
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		30*time.Second,
		informers.WithNamespace(serverNamespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()
		}),
	)
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
			names = append(names, d.Name)
		}
	}
	log.Debug().Msgf("Found %d deployments in cache", len(names))
	return names
}

func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
