package ctrl

import (
	"bytes"
	context "context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	JaegerNginxProxyV1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
)

type JaegerNginxProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func GenerateNginxConfig(nginxProxy *JaegerNginxProxyV1alpha0.JaegerNginxProxy) string {
	var config strings.Builder

	// Log format
	config.WriteString("log_format custom_format '$remote_addr - $remote_user [$time_local] '\n")
	config.WriteString("                             '\"$request\" \"args=$args\" \"q=$query_string\" '\n")
	config.WriteString("                             '\"url=$uri\" \"status=$status\" '\n")
	config.WriteString("                             '\"bytes=$body_bytes_sent\" \"ref=$http_referer\" '\n")
	config.WriteString("                             '\"agent=$http_user_agent\" \"$http_x_forwarded_for\" ';\n\n")

	// Upstream blocks
	for _, port := range nginxProxy.Spec.Ports {
		config.WriteString(fmt.Sprintf("upstream jaeger-collector-%s {\n", port.Name))
		config.WriteString(fmt.Sprintf("  server %s:%d;\n", nginxProxy.Spec.Upstream.CollectorHost, port.Port))
		config.WriteString("}\n\n")
	}

	// Server block
	config.WriteString("server {\n")
	config.WriteString(fmt.Sprintf("  listen %d default_server;\n\n", nginxProxy.Spec.ContainerPort))

	config.WriteString("  access_log /dev/stdout custom_format;\n")
	config.WriteString("  error_log  /dev/stderr;\n\n")

	config.WriteString("  proxy_connect_timeout 600;\n")
	config.WriteString("  proxy_send_timeout 600;\n")
	config.WriteString("  proxy_read_timeout 600;\n")
	config.WriteString("  send_timeout 600;\n")
	config.WriteString("  client_max_body_size 100m;\n\n")

	config.WriteString("  location /healthz {\n")
	config.WriteString("        access_log off;\n")
	config.WriteString("        return 200;\n")
	config.WriteString("  }\n\n")

	// Location blocks
	for _, port := range nginxProxy.Spec.Ports {
		config.WriteString(fmt.Sprintf("  location %s {\n", port.Path))
		config.WriteString(fmt.Sprintf("     proxy_pass http://jaeger-collector-%s;\n", port.Name))
		config.WriteString("  }\n\n")
	}

	config.WriteString("}\n")

	return config.String()
}

// ValidateNginxConfig performs basic validation of the nginx configuration
func ValidateNginxConfig(config string) error {
	// Basic syntax validation - check for common issues
	lines := strings.Split(config, "\n")

	// Check for balanced braces
	braceCount := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Count braces
		braceCount += strings.Count(line, "{")
		braceCount -= strings.Count(line, "}")

		// Check for unmatched braces
		if braceCount < 0 {
			return fmt.Errorf("unmatched closing brace on line %d: %s", i+1, line)
		}

		// Check for common syntax errors
		if strings.Contains(line, "server") && !strings.Contains(line, "{") && !strings.Contains(line, ";") {
			return fmt.Errorf("invalid server directive on line %d: %s", i+1, line)
		}

		if strings.Contains(line, "location") && !strings.Contains(line, "{") && !strings.Contains(line, ";") {
			return fmt.Errorf("invalid location directive on line %d: %s", i+1, line)
		}

		if strings.Contains(line, "upstream") && !strings.Contains(line, "{") && !strings.Contains(line, ";") {
			return fmt.Errorf("invalid upstream directive on line %d: %s", i+1, line)
		}
	}

	// Check for balanced braces at the end
	if braceCount != 0 {
		return fmt.Errorf("unmatched opening braces: %d unclosed", braceCount)
	}

	// Check for required directives
	if !strings.Contains(config, "server {") {
		return fmt.Errorf("missing server block")
	}

	if !strings.Contains(config, "listen") {
		return fmt.Errorf("missing listen directive")
	}

	// Try to validate with nginx -t if available (optional)
	if err := validateWithNginx(config); err != nil {
		log.Warn().Err(err).Msg("nginx validation failed, but continuing with basic validation")
		// Don't fail the reconciliation for nginx validation errors
		// as nginx might not be available in the controller environment
	}

	return nil
}

// validateWithNginx attempts to validate the config using nginx -t
func validateWithNginx(config string) error {
	// Create a temporary file with the config
	tmpFile, err := os.CreateTemp("", "nginx-config-*.conf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write config to temp file
	if _, err := tmpFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write config to temp file: %w", err)
	}
	tmpFile.Close()

	// Run nginx -t to validate
	cmd := exec.Command("nginx", "-t", "-c", tmpFile.Name())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nginx validation failed: %s", stderr.String())
	}

	return nil
}

func buildConfigMap(nginxProxy *JaegerNginxProxyV1alpha0.JaegerNginxProxy) (*corev1.ConfigMap, error) {
	config := GenerateNginxConfig(nginxProxy)

	// Validate the nginx configuration before creating the ConfigMap
	if err := ValidateNginxConfig(config); err != nil {
		return nil, fmt.Errorf("nginx configuration validation failed: %w", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxProxy.Name,
			Namespace: nginxProxy.Namespace,
		},
		Data: map[string]string{
			"proxy.conf": config,
		},
	}, nil
}

func buildDeployment(nginxProxy *JaegerNginxProxyV1alpha0.JaegerNginxProxy) *appsv1.Deployment {
	replicas := int32(nginxProxy.Spec.ReplicaCount)
	image := nginxProxy.Spec.Image.Repository + ":" + nginxProxy.Spec.Image.Tag
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxProxy.Name,
			Namespace: nginxProxy.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": nginxProxy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": nginxProxy.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: image,
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(nginxProxy.Spec.Resources.Limits.CPU),
								corev1.ResourceMemory: resource.MustParse(nginxProxy.Spec.Resources.Limits.Memory),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(nginxProxy.Spec.Resources.Requests.CPU),
								corev1.ResourceMemory: resource.MustParse(nginxProxy.Spec.Resources.Requests.Memory),
							},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "contents",
							MountPath: "/etc/nginx/conf.d",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "contents",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: nginxProxy.Name,
								},
							},
						},
					}},
				},
			},
		},
	}
}

func (r *JaegerNginxProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var page JaegerNginxProxyV1alpha0.JaegerNginxProxy
	err := r.Get(ctx, req.NamespacedName, &page)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// JaegerNginxProxy deleted: clean up resources
			log.Info().Msgf("JaegerNginxProxy deleted: %s %s", req.Name, req.Namespace)
			var cm corev1.ConfigMap
			cm.Name = req.Name
			cm.Namespace = req.Namespace
			_ = r.Delete(ctx, &cm) // ignore errors if not found
			var dep appsv1.Deployment
			dep.Name = req.Name
			dep.Namespace = req.Namespace
			_ = r.Delete(ctx, &dep)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 1. Ensure ConfigMap exists and is up to date
	cm, err := buildConfigMap(&page)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to build ConfigMap for JaegerNginxProxy: %s %s", page.Name, page.Namespace)
		return ctrl.Result{}, err
	}
	if err := ctrl.SetControllerReference(&page, cm, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	log.Info().Msgf("Reconciling ConfigMap for JaegerNginxProxy: %s %s", cm.Name, cm.Namespace)
	var existingCM corev1.ConfigMap

	if err := r.Get(ctx, req.NamespacedName, &existingCM); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info().Msgf("Creating ConfigMap for JaegerNginxProxy: %s %s", cm.Name, cm.Namespace)
		if err := r.Create(ctx, cm); err != nil {
			log.Error().Err(err).Msgf("Failed to create ConfigMap: %s %s", cm.Name, cm.Namespace)
			return ctrl.Result{}, err
		}
		log.Info().Msgf("Successfully created ConfigMap: %s %s", cm.Name, cm.Namespace)
	} else {
		// Check if ConfigMap data needs to be updated
		if !reflect.DeepEqual(existingCM.Data, cm.Data) {
			log.Info().Msgf("ConfigMap data changed, updating: %s %s", cm.Name, cm.Namespace)
			log.Debug().Interface("old_data", existingCM.Data).Interface("new_data", cm.Data).Msg("ConfigMap data comparison")

			existingCM.Data = cm.Data
			if err := r.Update(ctx, &existingCM); err != nil {
				if errors.IsConflict(err) {
					log.Info().Msgf("ConfigMap update conflict, requeuing: %s %s", cm.Name, cm.Namespace)
					// Requeue to try again with the latest version
					return ctrl.Result{Requeue: true}, nil
				}
				log.Error().Err(err).Msgf("Failed to update ConfigMap: %s %s", cm.Name, cm.Namespace)
				return ctrl.Result{}, err
			}
			log.Info().Msgf("Successfully updated ConfigMap: %s %s", cm.Name, cm.Namespace)
		} else {
			log.Debug().Msgf("ConfigMap is up to date: %s %s", cm.Name, cm.Namespace)
		}
	}

	// 2. Ensure Deployment exists and is up to date
	dep := buildDeployment(&page)
	if err := ctrl.SetControllerReference(&page, dep, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	log.Info().Msgf("Reconciling Deployment for JaegerNginxProxy: %s %s", dep.Name, dep.Namespace)
	var existingDep appsv1.Deployment

	if err := r.Get(ctx, req.NamespacedName, &existingDep); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, dep); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		updated := false

		if *existingDep.Spec.Replicas != *dep.Spec.Replicas {
			existingDep.Spec.Replicas = dep.Spec.Replicas
			updated = true
		}

		if existingDep.Spec.Template.Spec.Containers[0].Image != dep.Spec.Template.Spec.Containers[0].Image {
			existingDep.Spec.Template.Spec.Containers[0].Image = dep.Spec.Template.Spec.Containers[0].Image
			updated = true
		}

		if updated {
			if err := r.Update(ctx, &existingDep); err != nil {
				if errors.IsConflict(err) {
					// Requeue to try again with the latest version
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, err
			}
		}
	}

	// Improved status logic: check Deployment status
	var depToCheck appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &depToCheck); err != nil {
		page.Status.Ready = false
		page.Status.Message = "Deployment not found"
	} else {
		desired := int32(1)
		if depToCheck.Spec.Replicas != nil {
			desired = *depToCheck.Spec.Replicas
		}

		// Handle different scenarios for readiness
		if desired == 0 {
			// When desired replicas is 0, the deployment is ready if no pods are running
			if depToCheck.Status.AvailableReplicas == 0 {
				page.Status.Ready = true
				page.Status.Message = "Deployment scaled to 0 replicas"
			} else {
				page.Status.Ready = false
				page.Status.Message = fmt.Sprintf("Scaling down: %d pods still running, desired: 0", depToCheck.Status.AvailableReplicas)
			}
		} else if depToCheck.Status.AvailableReplicas == desired {
			// When desired > 0 and all replicas are available
			page.Status.Ready = true
			page.Status.Message = fmt.Sprintf("All %d pods are running", desired)
		} else {
			// When desired > 0 but not all replicas are available
			page.Status.Ready = false
			page.Status.Message = fmt.Sprintf("Available replicas: %d/%d, Ready replicas: %d, Unavailable replicas: %d", depToCheck.Status.AvailableReplicas, desired, depToCheck.Status.ReadyReplicas, depToCheck.Status.UnavailableReplicas)
		}
	}

	log.Info().Bool("ready", page.Status.Ready).Str("message", page.Status.Message).Msg("Setting CR status")

	if err := r.Status().Update(ctx, &page); err != nil {
		if errors.IsConflict(err) {
			// Requeue if there's a conflict
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error().Err(err).Msg("Failed to update status")
		return ctrl.Result{}, err
	}
	log.Info().Msg("Successfully updated CR status")

	return ctrl.Result{}, nil
}

func AddJaegerNginxProxyController(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&JaegerNginxProxyV1alpha0.JaegerNginxProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(&JaegerNginxProxyReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
}
