package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotModelPricingRoundtrip(t *testing.T) {
	groupID := int64(50)
	apiKey := &APIKey{
		ID:      82,
		UserID:  40,
		GroupID: &groupID,
		Name:    "model-pricing-roundtrip",
		Status:  StatusActive,
		User: &User{
			ID:     40,
			Status: StatusActive,
		},
		Group: &Group{
			ID:                        groupID,
			Name:                      "priced-group",
			Platform:                  PlatformOpenAI,
			Status:                    StatusActive,
			LongContextPricingEnabled: true,
			ModelPricing: []ChannelModelPricing{{
				Platform:   PlatformOpenAI,
				Models:     []string{"gpt-5"},
				InputPrice: float64Ptr(2.5),
			}},
		},
	}

	svc := &APIKeyService{}
	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)
	require.NotNil(t, snapshot)
	require.Equal(t, 20, snapshot.Version)

	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: snapshot})
	require.NoError(t, err)
	var restored APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &restored))

	materialized, used, err := svc.applyAuthCacheEntry("sk-test", &restored)
	require.NoError(t, err)
	require.True(t, used)
	require.NotNil(t, materialized.Group)
	require.True(t, materialized.Group.LongContextPricingEnabled)
	require.Equal(t, apiKey.Group.ModelPricing, materialized.Group.ModelPricing)
}
