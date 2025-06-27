package kelm

import (
	"context"
	"os"

	"kelm-operator/internal/pkg/timer"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Init() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		logrus.Errorf("Failed to build kubeconfig: %v\n", err)
		os.Exit(1)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorf("Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	namespaces, err := getNamespaces(client, labels.Set{"kelm.riftonix.io/managed": "true"})
	if err != nil {
		logrus.Errorf("Failed to get namespaces: %v", err)
		os.Exit(1)
	}

	logrus.Infof("Found %d managed namespaces:\n", len(namespaces))
	for _, ns := range namespaces {
		removalTtl := ns.Annotations["kelm.riftonix.io/ttl.removal"]
		remainingTtl, err := timer.GetTimeUntilRemoval(ns.CreationTimestamp.Time, removalTtl)
		if err != nil {
			logrus.Errorf("Failed to parse annotations: %v\n", err)
			os.Exit(1)
		}
		entityAge := timer.GetEntityAge(ns.CreationTimestamp.Time)
		logrus.WithFields(logrus.Fields{
			"namespace":   ns.Name,
			"created":     ns.CreationTimestamp.Time.String(),
			"age":         entityAge,
			"ttl":         remainingTtl.String(),
			"annotations": ns.Annotations,
			"labels":      ns.Labels,
		}).Debug("Namespaces info:")
		ctx, _ := context.WithCancel(context.Background())
		go timer.CreateCountdown(ctx, ns.Name, int(remainingTtl.Seconds()))
	}
	select {}
}
