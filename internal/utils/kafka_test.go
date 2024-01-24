package utils

import (
	"strconv"
	"testing"
	"time"
)

func TestKafkaIsHealthy(t *testing.T) {
	testCases := []struct {
		brokerAddress  string
		topic          string
		expectedResult bool
	}{
		{"127.0.0.1:9093", "crypto-prices", true},
		{"broker2:9090", "topic2", false},
		{"broker3:9090", "topic3", false},
	}

	for _, tc := range testCases {
		t.Run(tc.brokerAddress+"_"+tc.topic, func(t *testing.T) {
			result := KafkaIsHealthy(tc.brokerAddress, tc.topic)
			if result != tc.expectedResult {
				t.Errorf("Expected KafkaIsHealthy(%s, %s) to be %v, but got %v", tc.brokerAddress, tc.topic, tc.expectedResult, result)
			}
		})
	}
}

func TestExponentialBackoff(t *testing.T) {
	testCases := []struct {
		attempt        int
		expectedResult time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
	}

	for _, tc := range testCases {
		t.Run("Attempt_"+strconv.Itoa(tc.attempt), func(t *testing.T) {
			duration := ExponentialBackoff(tc.attempt)
			if duration != tc.expectedResult {
				t.Errorf("Expected ExponentialBackoff(%d) to be %v, but got %v", tc.attempt, tc.expectedResult, duration)
			}
		})
	}
}
