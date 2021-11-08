package cluster

import (
	"context"
	"fmt"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetApplicationClusterConfig(ctx context.Context, c client.Client, app *cdv1.Application) (*rest.Config, error) {
	clusterSecret, err := getDestClusterSecret(ctx, c, app.Spec.Destination.Name, app.Namespace)
	if err != nil {
		return nil, err
	}
	clusterConfig, err := secretToClusterConfig(clusterSecret)
	if err != nil {
		return nil, err
	}
	return clusterConfig, err
}

func getDestClusterSecret(ctx context.Context, c client.Client, destName, appNamespace string) (*corev1.Secret, error) {
	secretName := destName + "-kubeconfig"
	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
	}

	if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: appNamespace}, clusterSecret); err != nil {
		return nil, fmt.Errorf("unable to find cluster secret %s: %v", secretName, err)
	}
	return clusterSecret, nil
}

// secretToCluster converts a secret into a Cluster object
func secretToClusterConfig(s *corev1.Secret) (*rest.Config, error) {
	kubeconfig := s.Data["value"]

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}
