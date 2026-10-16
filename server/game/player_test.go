package game

import (
	"errors"
	"strings"
	"testing"
)

func TestPlayerManager_AddPlayer_Validation(t *testing.T) {
	pm := NewPlayerManager()

	tests := []struct {
		name     string
		playerID string
		wantErr  error
	}{
		{
			name:     "valid simple id",
			playerID: "alice",
			wantErr:  nil,
		},
		{
			name:     "valid id with numbers and underscores",
			playerID: "user_123",
			wantErr:  nil,
		},
		{
			name:     "empty player id",
			playerID: "",
			wantErr:  ErrInvalidPlayerID,
		},
		{
			name:     "whitespace-only player id",
			playerID: "    ",
			wantErr:  ErrInvalidPlayerID,
		},
		{
			name:     "too long player id (>32 runes)",
			playerID: strings.Repeat("a", 33),
			wantErr:  ErrInvalidPlayerID,
		},
		{
			name:     "player id with control characters (newline)",
			playerID: "alice\nbob",
			wantErr:  ErrInvalidPlayerID,
		},
		{
			name:     "player id with null byte",
			playerID: "alice\x00",
			wantErr:  ErrInvalidPlayerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pm.AddPlayer(tt.playerID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("AddPlayer(%q) error = %v, want %v", tt.playerID, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("AddPlayer(%q) unexpected error = %v", tt.playerID, err)
				}
			}
		})
	}
}

func TestPlayerManager_DuplicatePlayerFails(t *testing.T) {
	pm := NewPlayerManager()

	if _, err := pm.AddPlayer("player-one"); err != nil {
		t.Fatalf("first AddPlayer error = %v", err)
	}

	if _, err := pm.AddPlayer("player-one"); err == nil {
		t.Fatal("second AddPlayer with same id should have failed")
	}
}

func TestPlayerManager_AuthenticatePlayer(t *testing.T) {
	pm := NewPlayerManager()

	snap, err := pm.AddPlayer("player-auth")
	if err != nil {
		t.Fatalf("AddPlayer error = %v", err)
	}

	token, ok := pm.SessionTokenForPlayer(snap.ID)
	if !ok || token == "" {
		t.Fatalf("SessionTokenForPlayer returned empty/false")
	}

	// Valid token
	if _, authOK := pm.AuthenticatePlayer(snap.ID, token); !authOK {
		t.Fatalf("AuthenticatePlayer with valid token failed")
	}

	// Invalid token
	if _, authOK := pm.AuthenticatePlayer(snap.ID, "wrong-token"); authOK {
		t.Fatalf("AuthenticatePlayer with wrong token succeeded, want failure")
	}

	// Non-existent player
	if _, authOK := pm.AuthenticatePlayer("ghost", token); authOK {
		t.Fatalf("AuthenticatePlayer with unknown player succeeded, want failure")
	}
}
