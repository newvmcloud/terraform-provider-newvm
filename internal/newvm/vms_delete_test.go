package newvm

import (
	"testing"
	"time"
)

func TestFormatOrderEndDate(t *testing.T) {
	timezone, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	now := time.Date(2026, time.May, 29, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		billedUntil string
		expected    string
		wantErr     bool
	}{
		{
			name:        "empty billed until falls back to now",
			billedUntil: "",
			expected:    "2026-05-29",
		},
		{
			name:        "supports RFC3339 milliseconds",
			billedUntil: "2026-05-30T23:59:59.000Z",
			expected:    "2026-05-31",
		},
		{
			name:        "supports RFC3339 without milliseconds",
			billedUntil: "2026-05-30T23:59:59Z",
			expected:    "2026-05-31",
		},
		{
			name:        "supports date only",
			billedUntil: "2026-05-30",
			expected:    "2026-05-30",
		},
		{
			name:        "invalid billed until returns error",
			billedUntil: "not-a-date",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := formatOrderEndDate(tc.billedUntil, now, timezone)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, got)
			}
		})
	}
}
