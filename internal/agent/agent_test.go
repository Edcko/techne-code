package agent

import (
	"encoding/json"
	"testing"
)

func TestToolCallHash(t *testing.T) {
	hash1 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/a.go"}`))
	hash2 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/a.go"}`))
	hash3 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/b.go"}`))

	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}
}

func TestDetectLoop_NoLoop(t *testing.T) {
	hashes := []string{"a", "b", "c", "d", "e"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop with diverse hashes")
	}
}

func TestDetectLoop_Loop(t *testing.T) {
	// Same hash repeated 6 times
	hashes := []string{"a", "a", "a", "a", "a", "a"}
	if !detectLoop(hashes, 5, 10) {
		t.Error("should detect loop with 6 identical hashes (maxCount=5)")
	}
}

func TestDetectLoop_BelowThreshold(t *testing.T) {
	// Same hash 5 times (exactly at threshold, should NOT trigger)
	hashes := []string{"a", "a", "a", "a", "a"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop when count equals maxCount")
	}
}

func TestDetectLoop_ShortHistory(t *testing.T) {
	hashes := []string{"a", "a"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop with short history")
	}
}

func TestDetectLoop_WindowSize(t *testing.T) {
	// 10 unique hashes, then 6 identical — should detect in window
	hashes := []string{
		"b", "b", "b", "b", "b", "b", "b", "b", "b", "b",
		"a", "a", "a", "a", "a", "a",
	}
	if !detectLoop(hashes, 5, 10) {
		t.Error("should detect loop within window")
	}
}

func TestToJSON(t *testing.T) {
	result := toJSON(map[string]string{"key": "value"})
	if string(result) != `{"key":"value"}` {
		t.Errorf("unexpected JSON: %s", result)
	}
}

func TestToJSON_Nil(t *testing.T) {
	result := toJSON(nil)
	if string(result) != "null" {
		t.Errorf("unexpected JSON for nil: %s", result)
	}
}
