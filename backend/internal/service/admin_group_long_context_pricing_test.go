//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateGroupLongContextPricingDefaultAndExplicitDisable(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{name: "omitted defaults to enabled", expected: true},
		{name: "explicit false is preserved", enabled: boolPtr(false), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &groupRepoStubForAdmin{}
			svc := &adminServiceImpl{groupRepo: repo}
			group, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
				Name:                      "pricing-group",
				Platform:                  PlatformOpenAI,
				RateMultiplier:            1,
				LongContextPricingEnabled: tt.enabled,
			})
			require.NoError(t, err)
			require.Equal(t, tt.expected, group.LongContextPricingEnabled)
		})
	}
}
