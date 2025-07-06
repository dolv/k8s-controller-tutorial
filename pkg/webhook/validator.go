package webhook

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	JaegerNginxProxyV1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
	ctrl "github.com/dolv/k8s-controller-tutorial/pkg/ctrl"
)

// JaegerNginxProxyValidator validates JaegerNginxProxy resources
type JaegerNginxProxyValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

//+kubebuilder:webhook:path=/validate-jaeger-nginx-proxy-platform-engineer-stream-v1alpha0-jaegernginxproxy,mutating=false,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1;v1beta1,groups=jaeger-nginx-proxy.platform-engineer.stream,resources=jaegernginxproxies,verbs=create;update,versions=v1alpha0,name=vjaegernginxproxy.kb.io

var _ webhook.CustomValidator = &JaegerNginxProxyValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *JaegerNginxProxyValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nginxProxy := obj.(*JaegerNginxProxyV1alpha0.JaegerNginxProxy)
	log.Info().Msgf("Validating creation of JaegerNginxProxy: %s/%s", nginxProxy.Namespace, nginxProxy.Name)

	err := v.validateJaegerNginxProxy(nginxProxy)
	return nil, err
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *JaegerNginxProxyValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newNginxProxy := newObj.(*JaegerNginxProxyV1alpha0.JaegerNginxProxy)

	log.Info().Msgf("Validating update of JaegerNginxProxy: %s/%s", newNginxProxy.Namespace, newNginxProxy.Name)

	err := v.validateJaegerNginxProxy(newNginxProxy)
	return nil, err
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *JaegerNginxProxyValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nginxProxy := obj.(*JaegerNginxProxyV1alpha0.JaegerNginxProxy)
	log.Info().Msgf("Validating deletion of JaegerNginxProxy: %s/%s", nginxProxy.Namespace, nginxProxy.Name)

	// Deletion is always allowed
	return nil, nil
}

// validateJaegerNginxProxy performs comprehensive validation of the JaegerNginxProxy resource
func (v *JaegerNginxProxyValidator) validateJaegerNginxProxy(nginxProxy *JaegerNginxProxyV1alpha0.JaegerNginxProxy) error {
	var allErrs field.ErrorList

	// Validate basic fields
	if nginxProxy.Spec.ReplicaCount <= 0 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "replicaCount"),
			nginxProxy.Spec.ReplicaCount,
			"replicaCount must be greater than 0",
		))
	}

	if nginxProxy.Spec.ContainerPort <= 0 || nginxProxy.Spec.ContainerPort > 65535 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "containerPort"),
			nginxProxy.Spec.ContainerPort,
			"containerPort must be between 1 and 65535",
		))
	}

	// Validate upstream
	if nginxProxy.Spec.Upstream.CollectorHost == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "upstream", "collectorHost"),
			"collectorHost is required",
		))
	}

	// Validate ports
	if len(nginxProxy.Spec.Ports) == 0 {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "ports"),
			"at least one port must be specified",
		))
	}

	portNames := make(map[string]bool)
	for i, port := range nginxProxy.Spec.Ports {
		if port.Name == "" {
			allErrs = append(allErrs, field.Required(
				field.NewPath("spec", "ports").Index(i).Child("name"),
				"port name is required",
			))
		} else if portNames[port.Name] {
			allErrs = append(allErrs, field.Duplicate(
				field.NewPath("spec", "ports").Index(i).Child("name"),
				port.Name,
			))
		} else {
			portNames[port.Name] = true
		}

		if port.Port <= 0 || port.Port > 65535 {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec", "ports").Index(i).Child("port"),
				port.Port,
				"port must be between 1 and 65535",
			))
		}

		if port.Path == "" {
			allErrs = append(allErrs, field.Required(
				field.NewPath("spec", "ports").Index(i).Child("path"),
				"port path is required",
			))
		}
	}

	// Validate image
	if nginxProxy.Spec.Image.Repository == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "image", "repository"),
			"image repository is required",
		))
	}

	if nginxProxy.Spec.Image.Tag == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "image", "tag"),
			"image tag is required",
		))
	}

	// Validate resources
	if nginxProxy.Spec.Resources.Limits.CPU == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "resources", "limits", "cpu"),
			"CPU limit is required",
		))
	}

	if nginxProxy.Spec.Resources.Limits.Memory == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "resources", "limits", "memory"),
			"memory limit is required",
		))
	}

	if nginxProxy.Spec.Resources.Requests.CPU == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "resources", "requests", "cpu"),
			"CPU request is required",
		))
	}

	if nginxProxy.Spec.Resources.Requests.Memory == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "resources", "requests", "memory"),
			"memory request is required",
		))
	}

	// Validate nginx configuration generation
	if len(allErrs) == 0 {
		if err := v.validateNginxConfigGeneration(nginxProxy); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec"),
				nginxProxy.Spec,
				fmt.Sprintf("nginx configuration validation failed: %v", err),
			))
		}
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("validation failed: %v", allErrs)
	}

	return nil
}

// validateNginxConfigGeneration validates that the nginx configuration can be generated successfully
func (v *JaegerNginxProxyValidator) validateNginxConfigGeneration(nginxProxy *JaegerNginxProxyV1alpha0.JaegerNginxProxy) error {
	// Generate the nginx configuration
	config := ctrl.GenerateNginxConfig(nginxProxy)

	// Validate the generated configuration
	return ctrl.ValidateNginxConfig(config)
}

// InjectDecoder injects the decoder.
func (v *JaegerNginxProxyValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
