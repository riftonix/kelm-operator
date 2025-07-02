package timer

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Variable ttlRemoval — string with format "360m", "24h" and so on
// Function returns 0s or current ttl
func GetDuration(creationTime time.Time, ttl string, factor float64) (time.Duration, error) {
	baseTtlDuration, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, err
	}
	ttlDuration := time.Duration(float64(baseTtlDuration) * factor)
	removalTime := creationTime.UTC().Add(ttlDuration)
	now := time.Now().UTC()

	if now.After(removalTime) || now.Equal(removalTime) {
		return 0, nil
	}
	return removalTime.Sub(now), nil
}

func GetEntityAge(creationTime time.Time) time.Duration {
	return time.Since(creationTime).Truncate(time.Second)
}

func ParseTime(input string) (time.Time, error) {
	var timestamp time.Time
	timestamp, err := time.Parse(time.RFC3339, input)
	if err != nil {
		return timestamp, err
	}
	return timestamp, nil
}

func GetMaxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}

func GetMaxDuration(a, b string) (string, error) {
	aDuration, err := time.ParseDuration(a)
	if err != nil {
		return "", err
	}
	bDuration, err := time.ParseDuration(b)
	if err != nil {
		return "", err
	}
	if aDuration > bDuration {
		return a, nil
	}
	return b, nil
}

func CreateCountdown(ctx context.Context, envName string, ttlSeconds int, scenario string) {
	if ttlSeconds <= 0 {
		logrus.Debugf("Env '%s' TTL expired for scenario %s!", envName, scenario)
		return
	}
	timer := time.NewTimer(time.Duration(ttlSeconds) * time.Second)
	defer timer.Stop() // Delayed timer cleanup

	select {
	case <-ctx.Done():
		// Timer canceled
		logrus.Debugf("Env '%s' TTL countdown cancelled for scenario %s.", envName, scenario)
		return
	case <-timer.C:
		// Env expired
		logrus.Debugf("Env '%s' TTL expired after %d seconds for scenario %s!", envName, ttlSeconds, scenario)
	}
}
