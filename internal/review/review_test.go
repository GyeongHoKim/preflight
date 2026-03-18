package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityRank(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"critical", 3},
		{"warning", 2},
		{"info", 1},
		{"unknown", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			assert.Equal(t, tt.want, SeverityRank(tt.severity))
		})
	}
}

func TestSeverityRankOrdering(t *testing.T) {
	assert.Greater(t, SeverityRank(SeverityCritical), SeverityRank(SeverityWarning))
	assert.Greater(t, SeverityRank(SeverityWarning), SeverityRank(SeverityInfo))
}
