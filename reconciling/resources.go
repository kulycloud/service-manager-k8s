package reconciling

import (
	"fmt"
	protoStorage "github.com/kulycloud/protocol/storage"
	"github.com/kulycloud/service-manager-k8s/config"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func serviceDeploymentName(name *protoStorage.NamespacedName) string {
	return fmt.Sprintf("svc-%s-%s", name.Namespace, name.Name)
}

func serviceLBDeploymentName(name *protoStorage.NamespacedName) string {
	return fmt.Sprintf("svclb-%s-%s", name.Namespace, name.Name)
}

func pullSecretName(name *protoStorage.NamespacedName) string {
	return fmt.Sprintf("svc-%s-%s-pullsecret", name.Namespace, name.Name)
}

func buildPullSecrets(name *protoStorage.NamespacedName, service *protoStorage.Service) *corev1.Secret {
	data := []byte(service.PullSecrets)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pullSecretName(name),
			Namespace: config.GlobalConfig.ServiceNamespace,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": data,
		},
	}
}

func buildDeploymentFromService(name *protoStorage.NamespacedName, service *protoStorage.Service) *appsv1.Deployment {
	envVars := make([]corev1.EnvVar, 0)
	for name, value := range service.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	replicas := int32(service.Replicas)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceDeploymentName(name),
			Namespace: config.GlobalConfig.ServiceNamespace,
			Labels: map[string]string{
				namespaceLabel: name.Namespace,
				typeLabel:      typeLabelService,
				nameLabel:      name.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					namespaceLabel: name.Namespace,
					typeLabel:      typeLabelService,
					nameLabel:      name.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						namespaceLabel: name.Namespace,
						typeLabel:      typeLabelService,
						nameLabel:      name.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app-container",
							Image: service.Image,
							Args:  service.Arguments,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http-port",
									ContainerPort: int32(config.GlobalConfig.HTTPPort),
								},
							},
							Env: envVars,
						},
					},
				},
			},
		},
	}

	if service.PullSecrets != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", config.GlobalConfig.ServiceNamespace, pullSecretName(name)),
			},
		}
	}

	return &deployment
}

func buildLoadBalancerDeploymentFromService(name *protoStorage.NamespacedName, _ *protoStorage.Service) *appsv1.Deployment {
	var replicas int32 = 2
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceLBDeploymentName(name),
			Namespace: config.GlobalConfig.ServiceNamespace,
			Labels: map[string]string{
				namespaceLabel: name.Namespace,
				typeLabel:      typeLabelLB,
				nameLabel:      name.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					namespaceLabel: name.Namespace,
					typeLabel:      typeLabelLB,
					nameLabel:      name.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						namespaceLabel: name.Namespace,
						typeLabel:      typeLabelLB,
						nameLabel:      name.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "lb-container",
							Image:           config.GlobalConfig.LoadBalancerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http-port",
									ContainerPort: int32(config.GlobalConfig.HTTPPort),
								},
								{
									Name:          "control-port",
									ContainerPort: int32(config.GlobalConfig.LoadBalancerControlPort),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "PORT",
									Value: strconv.FormatInt(int64(config.GlobalConfig.LoadBalancerControlPort), 10),
								},
								{
									Name: "HTTP_PORT",
									Value: strconv.FormatInt(int64(config.GlobalConfig.HTTPPort), 10),
								},
							},
						},
					},
				},
			},
		},
	}
}
