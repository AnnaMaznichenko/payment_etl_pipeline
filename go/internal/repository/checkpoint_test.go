package repository

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/source"
)

func TestCheckpointRepository_Get(t *testing.T) {
	t.Run("returns zero values for non-existent sources", testCheckpointGetNonExistent)
	t.Run("returns existing checkpoints", testCheckpointGetExisting)
	t.Run("mixed existing and non-existing", testCheckpointGetMixed)
}

func testCheckpointGetNonExistent(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	sources := []source.Source{source.Cards, source.Wallets, source.Crypto}

	result, err := repo.Get(context.Background(), sources)
	require.NoError(t, err)

	assert.Len(t, result, len(sources))

	for _, s := range sources {
		ts, exists := result[s]
		assert.True(t, exists, "source %s should be in result", s)
		assert.True(t, ts.IsZero(), "source %s should have zero time", s)
	}
}

func testCheckpointGetExisting(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	cardsTime := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Millisecond)
	walletsTime := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Millisecond)

	checkpoints := []dbo.ETLCheckpoint{
		{SourceName: source.Cards.String(), LastProcessedAt: &cardsTime},
		{SourceName: source.Wallets.String(), LastProcessedAt: &walletsTime},
	}
	require.NoError(t, testMetaDB.Create(&checkpoints).Error)

	sources := []source.Source{source.Cards, source.Wallets}
	result, err := repo.Get(context.Background(), sources)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, cardsTime, result[source.Cards])
	assert.Equal(t, walletsTime, result[source.Wallets])
}

func testCheckpointGetMixed(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	cardsTime := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Millisecond)
	checkpoint := dbo.ETLCheckpoint{
		SourceName:      source.Cards.String(),
		LastProcessedAt: &cardsTime,
	}
	require.NoError(t, testMetaDB.Create(&checkpoint).Error)

	sources := []source.Source{source.Cards, source.Crypto}
	result, err := repo.Get(context.Background(), sources)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, cardsTime, result[source.Cards])
	assert.True(t, result[source.Crypto].IsZero(), "crypto should have zero time")
}

func TestCheckpointRepository_Update(t *testing.T) {
	t.Run("inserts new checkpoints", testCheckpointUpdateInsert)
	t.Run("updates existing checkpoints", testCheckpointUpdateUpsert)
	t.Run("empty map is no-op", testCheckpointUpdateEmpty)
}

func testCheckpointUpdateInsert(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	cardsTime := time.Now().UTC().Truncate(time.Millisecond)
	walletsTime := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Millisecond)

	updates := map[source.Source]time.Time{
		source.Cards:   cardsTime,
		source.Wallets: walletsTime,
	}

	require.NoError(t, repo.Update(context.Background(), updates))

	var checkpoints []dbo.ETLCheckpoint
	require.NoError(t, testMetaDB.Order("source_name").Find(&checkpoints).Error)
	require.Len(t, checkpoints, 2)

	assert.Equal(t, source.Cards.String(), checkpoints[0].SourceName)
	assert.Equal(t, cardsTime, *checkpoints[0].LastProcessedAt)

	assert.Equal(t, source.Wallets.String(), checkpoints[1].SourceName)
	assert.Equal(t, walletsTime, *checkpoints[1].LastProcessedAt)
}

func testCheckpointUpdateUpsert(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	ctx := context.Background()

	firstTime := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Millisecond)
	updates1 := map[source.Source]time.Time{
		source.Cards: firstTime,
	}
	require.NoError(t, repo.Update(ctx, updates1))

	secondTime := time.Now().UTC().Truncate(time.Millisecond)
	updates2 := map[source.Source]time.Time{
		source.Cards: secondTime,
	}
	require.NoError(t, repo.Update(ctx, updates2))

	var checkpoints []dbo.ETLCheckpoint
	require.NoError(t, testMetaDB.Find(&checkpoints).Error)
	require.Len(t, checkpoints, 1, "should have only one checkpoint after upsert")

	assert.Equal(t, source.Cards.String(), checkpoints[0].SourceName)
	assert.Equal(t, secondTime, *checkpoints[0].LastProcessedAt, "should have updated time")
}

func testCheckpointUpdateEmpty(t *testing.T) {
	t.Cleanup(truncateCheckpoints)
	repo := NewCheckpointRepository(testMetaDB)

	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, map[source.Source]time.Time{}))
	require.NoError(t, repo.Update(ctx, nil))

	var count int64
	require.NoError(t, testMetaDB.Model(&dbo.ETLCheckpoint{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func truncateCheckpoints() {
	if err := testMetaDB.Exec("TRUNCATE etl_checkpoints RESTART IDENTITY CASCADE").Error; err != nil {
		log.Printf("truncate checkpoints: %v", err)
	}
}
