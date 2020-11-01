package reconciling

import (
	"context"
	"fmt"
	protoStorage "github.com/kulycloud/protocol/storage"
	"github.com/kulycloud/service-manager-k8s/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) ReconcileDeployments(ctx context.Context, namespace string) error {
	// Storage should be ready
	serviceNames, err := r.storage.GetServicesInNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	deploymentsClient := r.clientset.AppsV1().Deployments(config.GlobalConfig.ServiceNamespace)
	secretsClient := r.clientset.CoreV1().Secrets(config.GlobalConfig.ServiceNamespace)

	deployments, err := deploymentsClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s=%s", typeLabel, typeLabelService, namespaceLabel, namespace)})
	if err != nil {
		return err
	}

	updated := make(map[string]bool)
	for _, dep := range deployments.Items {
		updated[dep.Name] = false
	}

	for _, name := range serviceNames {
		namespacedName := &protoStorage.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		serviceName := serviceDeploymentName(namespacedName)
		_, existing := updated[serviceName]
		updated[serviceName] = true

		service, err := r.storage.GetService(ctx, namespace, name)
		if err != nil {
			continue
		}

		if service.PullSecrets != "" {
			pullSecrets := buildPullSecrets(namespacedName, service)
			if existing {
				_, err = secretsClient.Update(ctx, pullSecrets, metav1.UpdateOptions{})
			} else {
				_, err = secretsClient.Create(ctx, pullSecrets, metav1.CreateOptions{})
			}
			if err != nil {
				logger.Warnw("Could not update/create PullSecret", "err", err, "existing", existing, "namespacedName", namespacedName)
				continue
			}
		}

		deployment := buildDeploymentFromService(namespacedName, service)

		if existing {
			_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		} else {
			_, err = deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
		}
		if err != nil {
			logger.Warnw("Could not update/create Deployment", "err", err, "existing", existing, "namespacedName", namespacedName)
			continue
		}

		loadbalancer := buildLoadBalancerDeploymentFromService(namespacedName, service)
		if existing {
			_, err = deploymentsClient.Update(ctx, loadbalancer, metav1.UpdateOptions{})
		} else {
			_, err = deploymentsClient.Create(ctx, loadbalancer, metav1.CreateOptions{})
		}
		if err != nil {
			logger.Warnw("Could not update/create LoadBalancer", "err", err, "existing", existing, "namespacedName", namespacedName)
			continue
		}

	}

	for name, handled := range updated {
		if handled == true {
			continue
		}

		// delete service that no longer exists
		namespacedName := &protoStorage.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := deploymentsClient.Delete(ctx, serviceDeploymentName(namespacedName), metav1.DeleteOptions{})
		if err != nil {
			logger.Warnw("Could not delete Deployment", "err", err, "namespacedName", namespacedName)
		}

		err = deploymentsClient.Delete(ctx, serviceLBDeploymentName(namespacedName), metav1.DeleteOptions{})
		if err != nil {
			logger.Warnw("Could not delete LoadBalancer", "err", err, "namespacedName", namespacedName)
		}

		// This will probably throw errors quite often. Still cheaper than to check whether there are pull secrets before
		_ = secretsClient.Delete(ctx, pullSecretName(namespacedName), metav1.DeleteOptions{})
	}

	return nil
}
