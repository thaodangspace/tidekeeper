package player

import (
	"context"
	"testing"
)

type readerStub struct {
	gotID   ID
	me      Me
	unlocks []KeeperUnlock
	err     error
}

func (r *readerStub) GetMe(_ context.Context, playerID ID) (Me, error) {
	r.gotID = playerID
	return r.me, r.err
}

func (r *readerStub) ListUnlocks(_ context.Context, playerID ID) ([]KeeperUnlock, error) {
	r.gotID = playerID
	return r.unlocks, r.err
}

func TestServiceListUnlocks(t *testing.T) {
	reader := &readerStub{unlocks: []KeeperUnlock{{DefinitionKey: "harbor_warden", DefinitionVersion: 1, UnlockSource: "SHOP"}}}
	service := NewService(reader)

	unlocks, err := service.ListUnlocks(context.Background(), ID("player-db-id"))
	if err != nil {
		t.Fatalf("ListUnlocks() error = %v", err)
	}
	if reader.gotID != ID("player-db-id") {
		t.Errorf("reader player ID = %q, want player-db-id", reader.gotID)
	}
	if len(unlocks) != 1 || unlocks[0].DefinitionKey != "harbor_warden" {
		t.Errorf("ListUnlocks() = %+v", unlocks)
	}
}

func TestServiceGetMe(t *testing.T) {
	activeVoyageID := "voy_test"
	reader := &readerStub{me: Me{PublicID: "plr_test", ActiveVoyageID: &activeVoyageID}}
	service := NewService(reader)

	got, err := service.GetMe(context.Background(), ID("player-db-id"))
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if reader.gotID != ID("player-db-id") {
		t.Errorf("reader player ID = %q, want player-db-id", reader.gotID)
	}
	if got.PublicID != "plr_test" || got.ActiveVoyageID == nil || *got.ActiveVoyageID != activeVoyageID {
		t.Errorf("GetMe() = %+v", got)
	}
}
