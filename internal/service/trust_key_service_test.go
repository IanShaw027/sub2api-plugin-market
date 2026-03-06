package service

import (
	"context"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/trustkey"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func TestTrustKeyService_ListTrustKeys_All(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:trustkey_svc_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	for _, kID := range []string{"key-official", "key-community"} {
		kt := trustkey.KeyTypeOfficial
		if kID == "key-community" {
			kt = trustkey.KeyTypeCommunity
		}
		_, err := client.TrustKey.Create().
			SetKeyID(kID).
			SetPublicKey("pubkey-" + kID).
			SetKeyType(kt).
			SetOwnerName("owner").
			SetOwnerEmail("o@test.com").
			SetIsActive(true).
			Save(ctx)
		require.NoError(t, err)
	}

	repo := repository.NewTrustKeyRepository(client)
	svc := NewTrustKeyService(repo)

	keys, err := svc.ListTrustKeys(ctx, &ListTrustKeysRequest{})
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestTrustKeyService_ListTrustKeys_FilterByType(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:trustkey_svc_filter_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	for i, kt := range []trustkey.KeyType{trustkey.KeyTypeOfficial, trustkey.KeyTypeCommunity, trustkey.KeyTypeCommunity} {
		_, err := client.TrustKey.Create().
			SetKeyID(kt.String() + "-" + string(rune('0'+i))).
			SetPublicKey("pk").
			SetKeyType(kt).
			SetOwnerName("owner").
			SetOwnerEmail("o@test.com").
			SetIsActive(true).
			Save(ctx)
		require.NoError(t, err)
	}

	repo := repository.NewTrustKeyRepository(client)
	svc := NewTrustKeyService(repo)

	keys, err := svc.ListTrustKeys(ctx, &ListTrustKeysRequest{KeyType: "community"})
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestTrustKeyService_ListTrustKeys_FilterByActive(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:trustkey_svc_active_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	_, err := client.TrustKey.Create().
		SetKeyID("active-key").SetPublicKey("pk").
		SetKeyType(trustkey.KeyTypeOfficial).SetOwnerName("owner").SetOwnerEmail("o@test.com").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	_, err = client.TrustKey.Create().
		SetKeyID("inactive-key").SetPublicKey("pk").
		SetKeyType(trustkey.KeyTypeOfficial).SetOwnerName("owner").SetOwnerEmail("o@test.com").
		SetIsActive(false).Save(ctx)
	require.NoError(t, err)

	repo := repository.NewTrustKeyRepository(client)
	svc := NewTrustKeyService(repo)

	isActive := true
	keys, err := svc.ListTrustKeys(ctx, &ListTrustKeysRequest{IsActive: &isActive})
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "active-key", keys[0].KeyID)
}

func TestTrustKeyService_GetTrustKeyDetail(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:trustkey_svc_detail_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	_, err := client.TrustKey.Create().
		SetKeyID("detail-key").SetPublicKey("pk").
		SetKeyType(trustkey.KeyTypeOfficial).SetOwnerName("owner").SetOwnerEmail("o@test.com").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	repo := repository.NewTrustKeyRepository(client)
	svc := NewTrustKeyService(repo)

	key, err := svc.GetTrustKeyDetail(ctx, "detail-key")
	require.NoError(t, err)
	assert.Equal(t, "detail-key", key.KeyID)
}

func TestTrustKeyService_GetTrustKeyDetail_NotFound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:trustkey_svc_notfound_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := repository.NewTrustKeyRepository(client)
	svc := NewTrustKeyService(repo)

	_, err := svc.GetTrustKeyDetail(context.Background(), "nonexistent")
	assert.Error(t, err)
}
