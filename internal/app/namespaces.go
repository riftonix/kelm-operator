package kelm

import (
	"context"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func getNamespaces(client *kubernetes.Clientset, labelsSet labels.Set) ([]core.Namespace, error) {
	filter := meta.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelsSet).String(),
	}
	namespaces, err := client.CoreV1().Namespaces().List(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	return namespaces.Items, nil
}
