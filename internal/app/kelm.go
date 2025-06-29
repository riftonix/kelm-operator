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

type CountdownCancel struct {
	envName string
	ttl     int
	cancel  context.CancelFunc
}

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

	envs, err := getEnvs(client, labels.Set{"kelm.riftonix.io/managed": "true"})
	if err != nil {
		logrus.Errorf("Failed to get namespaces: %v", err)
		os.Exit(1)
	}
	countdowns := make([]CountdownCancel, 0)
	for envName, env := range envs {
		for _, ns := range env.Namespaces {
			entityAge := timer.GetEntityAge(ns.CreationTimestamp.Time)
			logrus.WithFields(logrus.Fields{
				"namespace":                 ns.Name,
				"created":                   ns.CreationTimestamp.Time.String(),
				"age":                       entityAge,
				"ttl":                       env.RemainingTtl,
				"annotations":               ns.Annotations,
				"labels":                    ns.Labels,
				"replenishRatio":            env.ReplenishRatio,
				"RemainingNotificationsTtl": env.RemainingNotificationsTtl,
			}).Debug("Namespaces info:")
		}
		ctx, cancel := context.WithCancel(context.Background())
		countdowns = append(countdowns, CountdownCancel{
			envName: envName,
			cancel:  cancel,
			ttl:     int(env.RemainingTtl.Seconds()),
		})
		go timer.CreateCountdown(ctx, envName, int(env.RemainingTtl.Seconds()), "removal")
		for _, remainingNotificationTtl := range env.RemainingNotificationsTtl {
			go timer.CreateCountdown(ctx, envName, int(remainingNotificationTtl.Seconds()), "notification")
		}
	}
	select {}
}
