package api

import (

    "net/http/httptest"
    "testing"
)

func TestValidateRoomAndPlayerIDs(t *testing.T) {
    tests := []struct {
        name       string
        roomID     string
        playerID   string
        wantValid  bool
    }{
        {"valid IDs", "room_01", "player-01", true},
        {"invalid room", "bad!room", "player01", false},
        {"invalid player", "room01", "bad@player", false},
        {"empty IDs", "", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            got := validateRoomAndPlayerIDs(w, tt.roomID, tt.playerID)
            if got != tt.wantValid {
                t.Fatalf("expected %v, got %v, body=%s", tt.wantValid, got, w.Body.String())
            }
        })
    }
}
