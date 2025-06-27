package timer

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Variable ttlRemoval — string with format "360m", "24h" and so on
// Function returns 0s or current ttl
func GetTimeUntilRemoval(created time.Time, ttlRemoval string) (time.Duration, error) {
	ttlDuration, err := time.ParseDuration(ttlRemoval)
	if err != nil {
		return 0, err
	}

	removalTime := created.Add(ttlDuration)
	now := time.Now()

	if now.After(removalTime) || now.Equal(removalTime) {
		return 0, nil
	}

	return removalTime.Sub(now), nil
}

func GetEntityAge(creationTime time.Time) time.Duration {
	return time.Since(creationTime).Truncate(time.Second)
}

func CreateCountdown(ctx context.Context, envName string, ttlSeconds int) {
	if ttlSeconds <= 0 {
		logrus.Debugf("Env %s TTL expired!\n", envName)
		return
	}
	timer := time.NewTimer(time.Duration(ttlSeconds) * time.Second)
	defer timer.Stop() // Delayed timer cleanup

	select {
	case <-ctx.Done():
		// Timer canceled
		logrus.Debugf("Env %s TTL countdown cancelled\n", envName)
		return
	case <-timer.C:
		// Env expired
		logrus.Debugf("Env %s TTL expired after %d seconds!\n", envName, ttlSeconds)
	}
}
