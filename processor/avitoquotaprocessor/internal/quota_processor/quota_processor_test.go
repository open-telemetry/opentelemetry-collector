package quota_processor

import (
	"testing"
	"time"
)

func TestTimestampToBucketName(t *testing.T) {
	// Mock current time for consistent testing
	// Note: The exact date doesn't matter as much as the relative calculations based on retentionDays
	now := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name              string
		retentionDays     int
		inputTime         time.Time
		expectedBucket    string
		expectedPartition string
	}{
		{
			name:              "1 day retention",
			retentionDays:     1,
			inputTime:         now,
			expectedBucket:    "7d",
			expectedPartition: "7d_2024-05-17",
		},
		{
			name:              "7 days retention",
			retentionDays:     7,
			inputTime:         now,
			expectedBucket:    "7d",
			expectedPartition: "7d_2024-05-23",
		},
		{
			name:              "8 days retention", // ttlBucketSizeDays = 1
			retentionDays:     8,
			inputTime:         now,
			expectedBucket:    "7d",
			expectedPartition: "7d_2024-05-24",
		},
		{
			name:              "9 days retention", // ttlBucketSizeDays = 2
			retentionDays:     9,
			inputTime:         now,
			expectedBucket:    "14d",
			expectedPartition: "14d_2024-05-25",
		},
		{
			name:              "13 days retention", // ttlBucketSizeDays = 2
			retentionDays:     13,
			inputTime:         now,
			expectedBucket:    "14d",
			expectedPartition: "14d_2024-05-29",
		},
		{
			name:              "14 days retention", // ttlBucketSizeDays = 2
			retentionDays:     14,
			inputTime:         now,
			expectedBucket:    "14d",
			expectedPartition: "14d_2024-05-31",
		},
		{
			name:              "15 days retention", // ttlBucketSizeDays = 2
			retentionDays:     15,
			inputTime:         now,
			expectedBucket:    "14d",
			expectedPartition: "14d_2024-05-31",
		},
		{
			name:              "16 days retention", // ttlBucketSizeDays = 3
			retentionDays:     16,
			inputTime:         now,
			expectedBucket:    "30d",
			expectedPartition: "30d_2024-06-01",
		},
		{
			name:              "32 days retention", // ttlBucketSizeDays = 3
			retentionDays:     32,
			inputTime:         now,
			expectedBucket:    "30d",
			expectedPartition: "30d_2024-06-19",
		},
		{
			name:              "33 days retention", // ttlBucketSizeDays = 9
			retentionDays:     33,
			inputTime:         now,
			expectedBucket:    "90d",
			expectedPartition: "90d_2024-06-25",
		},
		{
			name:              "90 days retention", // ttlBucketSizeDays = 9
			retentionDays:     90,
			inputTime:         now,
			expectedBucket:    "90d",
			expectedPartition: "90d_2024-08-18",
		},
		{
			name:              "100 days retention", // ttlBucketSizeDays = 9
			retentionDays:     100,
			inputTime:         now,
			expectedBucket:    "90d",
			expectedPartition: "90d_2024-08-27",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The function `timestampToBucketName` uses the input time `t` directly to calculate ttlBucketTimeUnixSeconds
			// It does not add retentionDays to `t` internally before this calculation.
			// The `ttlTimestamp` which is `time.Now().Add(retentionDays)` is passed to it from `GetInfraAttrsForRetention`.
			// So, for direct testing of `timestampToBucketName`, we should pass the already advanced time.
			ttlTimestamp := tc.inputTime.Add(time.Duration(tc.retentionDays) * 24 * time.Hour)
			bucketName, partitionName := timestampToBucketName(ttlTimestamp, tc.retentionDays)

			if bucketName != tc.expectedBucket {
				t.Errorf("Expected bucket name %s, got %s", tc.expectedBucket, bucketName)
			}
			if partitionName != tc.expectedPartition {
				t.Errorf("Expected partition name %s, got %s", tc.expectedPartition, partitionName)
			}
		})
	}
}

func TestGetInfraAttrsForRetention(t *testing.T) {
	fixedTestTime := time.Now()

	testCases := []struct {
		name                 string
		retentionDays        int
		expectedTTLTimestamp time.Time // Based on fixedTestTime
		expectedBucketName   string    // Based on production code's logic
	}{
		{
			name:                 "1 day retention",
			retentionDays:        1,
			expectedTTLTimestamp: fixedTestTime.Add(1 * 24 * time.Hour),
			expectedBucketName:   "7d", // Stays 7d due to current prod logic
		},
		{
			name:                 "7 days retention",
			retentionDays:        7,
			expectedTTLTimestamp: fixedTestTime.Add(7 * 24 * time.Hour),
			expectedBucketName:   "7d",
		},
		{
			name:                 "8 days retention",
			retentionDays:        8,
			expectedTTLTimestamp: fixedTestTime.Add(8 * 24 * time.Hour),
			expectedBucketName:   "7d", // Stays 7d
		},
		{
			name:                 "9 days retention",
			retentionDays:        9,
			expectedTTLTimestamp: fixedTestTime.Add(9 * 24 * time.Hour),
			expectedBucketName:   "14d", // Switches to 14d
		},
		{
			name:                 "15 days retention",
			retentionDays:        15,
			expectedTTLTimestamp: fixedTestTime.Add(15 * 24 * time.Hour),
			expectedBucketName:   "14d",
		},
		{
			name:                 "16 days retention",
			retentionDays:        16,
			expectedTTLTimestamp: fixedTestTime.Add(16 * 24 * time.Hour),
			expectedBucketName:   "30d",
		},
		{
			name:                 "32 days retention",
			retentionDays:        32,
			expectedTTLTimestamp: fixedTestTime.Add(32 * 24 * time.Hour),
			expectedBucketName:   "30d",
		},
		{
			name:                 "33 days retention",
			retentionDays:        33,
			expectedTTLTimestamp: fixedTestTime.Add(33 * 24 * time.Hour),
			expectedBucketName:   "90d",
		},
		{
			name:                 "90 days retention",
			retentionDays:        90,
			expectedTTLTimestamp: fixedTestTime.Add(90 * 24 * time.Hour),
			expectedBucketName:   "90d",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := GetInfraAttrsForRetention(tc.retentionDays)

			if attrs == nil {
				t.Fatalf("Expected InfraAttributes, got nil")
			}
			if !(attrs.TTLTimestamp.Year() == tc.expectedTTLTimestamp.Year() &&
				attrs.TTLTimestamp.Month() == tc.expectedTTLTimestamp.Month() &&
				attrs.TTLTimestamp.Day() == tc.expectedTTLTimestamp.Day() &&
				attrs.TTLTimestamp.Hour() == tc.expectedTTLTimestamp.Hour() &&
				(attrs.TTLTimestamp.Minute() >= tc.expectedTTLTimestamp.Minute()-1 && attrs.TTLTimestamp.Minute() <= tc.expectedTTLTimestamp.Minute()+1)) { // Allow +/- 1m
				t.Errorf("TTLTimestamp mismatch. Expected date/time around %v (UTC), got %v (UTC)",
					tc.expectedTTLTimestamp.Format(time.RFC3339), attrs.TTLTimestamp.In(time.UTC).Format(time.RFC3339))
			}

			if attrs.TTLBucketName != tc.expectedBucketName {
				t.Errorf("Expected TTLBucketName %s, got %s", tc.expectedBucketName, attrs.TTLBucketName)
			}

			_, expectedPartitionFromReturnedTTL := timestampToBucketName(attrs.TTLTimestamp, tc.retentionDays)
			if attrs.PartitionName != expectedPartitionFromReturnedTTL {
				t.Errorf("Returned PartitionName %s is not consistent with what timestampToBucketName would produce for returned TTLTimestamp %v (which is %s)",
					attrs.PartitionName, attrs.TTLTimestamp.In(time.UTC).Format(time.RFC3339), expectedPartitionFromReturnedTTL)
			}
		})
	}
}
