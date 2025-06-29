package kelm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"kelm-operator/internal/pkg/timer"

	"github.com/sirupsen/logrus"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Env struct {
	Name                      string
	Namespaces                []core.Namespace
	RemainingTtl              time.Duration
	ReplenishRatio            float64
	RemainingNotificationsTtl []time.Duration
}

func handleNamespace(ns core.Namespace) (string, string, float64, []float64, error) {
	isManaged := ns.Labels["kelm.riftonix.io/managed"]
	envName := ns.Labels["kelm.riftonix.io/env.name"]
	ttl := ns.Annotations["kelm.riftonix.io/ttl.removal"]
	replenishRatio := ns.Annotations["kelm.riftonix.io/ttl.replenishRatio"]
	notificationFactors := ns.Annotations["kelm.riftonix.io/ttl.notificationFactors"]
	if isManaged != "true" {
		return "", "", 0, []float64{}, fmt.Errorf("Namespace %s label kelm.riftonix.io/managed is not true", ns.Name)
	}
	if envName == "" {
		return "", "", 0, []float64{}, fmt.Errorf("Namespace %s has empty label kelm.riftonix.io/env.name", ns.Name)
	}
	if ttl == "" {
		return "", "", 0, []float64{}, fmt.Errorf("Namespace %s has empty annotation kelm.riftonix.io/ttl.removal", ns.Name)
	}
	if replenishRatio == "" {
		return "", "", 0, []float64{}, fmt.Errorf("Namespace %s has empty annotation kelm.riftonix.io/ttl.replenishRatio", ns.Name)
	}
	parsedReplenishRatio, err := strconv.ParseFloat(replenishRatio, 8)
	if err != nil {
		return "", "", 0, []float64{}, fmt.Errorf("Failed to parse namespace %s annotation kelm.riftonix.io/ttl.replenishRatio '%s': %w", ns.Name, replenishRatio, err)
	}
	if notificationFactors == "" {
		return "", "", 0, []float64{}, fmt.Errorf("Namespace %s has empty annotation kelm.riftonix.io/ttl.notificationFactors", ns.Name)
	}
	var unmarshaledNotificationFactors []float64
	err = json.Unmarshal([]byte(notificationFactors), &unmarshaledNotificationFactors)
	if err != nil {
		return "", "", 0, []float64{}, fmt.Errorf("Failed to parse namespace %s annotation kelm.riftonix.io/ttl.notificationFactors '%s': %w", ns.Name, notificationFactors, err)
	}
	return envName, ttl, parsedReplenishRatio, unmarshaledNotificationFactors, nil
}

func getEnvs(client *kubernetes.Clientset, labelsSet labels.Set) (map[string]Env, error) {
	filter := meta.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelsSet).String(),
	}
	namespaces, err := client.CoreV1().Namespaces().List(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	envs := make(map[string]Env)
	for _, ns := range namespaces.Items {
		envName, ttl, replenishRatio, notificationFactors, err := handleNamespace(ns)
		if err != nil {
			logrus.Warningf("%v", err)
			continue
		}
		env := envs[envName]
		env.Namespaces = append(env.Namespaces, ns)
		remainingTtl, err := timer.GetDuration(ns.CreationTimestamp.Time, ttl, 1)
		if err != nil {
			logrus.Warningf("Failed to parse annotations in %s: %v\n", ns.Name, err)
			continue
		}
		env.RemainingTtl = max(env.RemainingTtl, remainingTtl)
		env.ReplenishRatio = replenishRatio
		for _, factor := range notificationFactors {
			remainingNotificationTtl, err := timer.GetDuration(ns.CreationTimestamp.Time, ttl, factor)
			if err != nil {
				logrus.Warningf("Failed to parse annotations in %s: %v\n", ns.Name, err)
				continue
			}
			env.RemainingNotificationsTtl = append(env.RemainingNotificationsTtl, remainingNotificationTtl)
		}
		slices.Sort(env.RemainingNotificationsTtl)
		slices.Compact(env.RemainingNotificationsTtl)
		envs[envName] = env
	}
	return envs, nil
}
