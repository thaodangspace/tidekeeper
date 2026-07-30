package auth

import (
	"context"
	"testing"
)

func TestPrincipalContext(t *testing.T) {
	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Fatal("PrincipalFromContext() found a principal in an empty context")
	}

	want := Principal{AccountID: "account-id", PlayerID: "player-id"}
	got, ok := PrincipalFromContext(WithPrincipal(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("PrincipalFromContext() = (%+v, %t), want (%+v, true)", got, ok, want)
	}
}
