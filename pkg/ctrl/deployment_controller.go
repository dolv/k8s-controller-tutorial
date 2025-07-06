package ctrl

import (
	context "context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Cache to store previous deployment states
	previousStates map[string]*appsv1.Deployment
	mutex          sync.RWMutex
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.With().Str("controller", "DeploymentReconciler").Logger()
	var dep appsv1.Deployment
	err := r.Get(ctx, req.NamespacedName, &dep)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info().Msgf("[CONTROLLER][Delete] Deployment deleted: %s/%s", req.Namespace, req.Name)
			// Remove from cache
			r.mutex.Lock()
			delete(r.previousStates, req.NamespacedName.String())
			r.mutex.Unlock()
			return ctrl.Result{}, nil
		}
		log.Error().Err(err).Msg("Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Get previous state
	r.mutex.RLock()
	previous := r.previousStates[req.NamespacedName.String()]
	r.mutex.RUnlock()

	eventType := "[CONTROLLER][Add]"
	changes := []string{}

	if previous != nil {
		eventType = "[CONTROLLER][Update]"

		// Compare meaningful fields
		if previous.Spec.Replicas != nil && dep.Spec.Replicas != nil {
			if *previous.Spec.Replicas != *dep.Spec.Replicas {
				changes = append(changes, fmt.Sprintf("replicas: %d -> %d", *previous.Spec.Replicas, *dep.Spec.Replicas))
			}
		}

		if len(previous.Spec.Template.Spec.Containers) > 0 && len(dep.Spec.Template.Spec.Containers) > 0 {
			if previous.Spec.Template.Spec.Containers[0].Image != dep.Spec.Template.Spec.Containers[0].Image {
				changes = append(changes, fmt.Sprintf("image: %s -> %s",
					previous.Spec.Template.Spec.Containers[0].Image,
					dep.Spec.Template.Spec.Containers[0].Image))
			}
		}

		if previous.Generation != dep.Generation {
			changes = append(changes, fmt.Sprintf("generation: %d -> %d", previous.Generation, dep.Generation))
		}
	}

	// Store current state for next comparison
	r.mutex.Lock()
	if r.previousStates == nil {
		r.previousStates = make(map[string]*appsv1.Deployment)
	}
	r.previousStates[req.NamespacedName.String()] = dep.DeepCopy()
	r.mutex.Unlock()

	if len(changes) > 0 {
		log.Info().Msgf("%s Deployment reconciled: %s/%s - Changes: %s",
			eventType, req.Namespace, req.Name, strings.Join(changes, ", "))
	} else {
		log.Info().Msgf("%s Deployment reconciled: %s/%s (no spec changes)",
			eventType, req.Namespace, req.Name)
	}

	return ctrl.Result{}, nil
}

func AddDeploymentController(mgr manager.Manager) error {
	r := &DeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
