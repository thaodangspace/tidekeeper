package player

import (
	"context"
	"testing"
)

type readerStub struct {
	gotID   ID
	me      Me
	keepers []Keeper
	err     error
}

func (r *readerStub) GetMe(_ context.Context, playerID ID) (Me, error) {
	r.gotID = playerID
	return r.me, r.err
}

func (r *readerStub) ListKeepers(_ context.Context, playerID ID) ([]Keeper, error) {
	r.gotID = playerID
	return r.keepers, r.err
}

func TestServiceListKeepers(t *testing.T) {
	reader := &readerStub{keepers: []Keeper{{PublicID: "kpr_test", DefinitionKey: "harbor_warden", Level: 1}}}
	service := NewService(reader)

	keepers, err := service.ListKeepers(context.Background(), ID("player-db-id"))
	if err != nil {
		t.Fatalf("ListKeepers() error = %v", err)
	}
	if reader.gotID != ID("player-db-id") {
		t.Errorf("reader player ID = %q, want player-db-id", reader.gotID)
	}
	if len(keepers) != 1 || keepers[0].PublicID != "kpr_test" {
		t.Errorf("ListKeepers() = %+v", keepers)
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
