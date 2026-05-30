package a2a

import (
	"encoding/json"
	"testing"
)

func TestDefaultAgentCard_ValidJSON(t *testing.T) {
	card := DefaultAgentCard("http://localhost:8080")
	b, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("failed to marshal agent card: %v", err)
	}
	if !json.Valid(b) {
		t.Fatal("agent card JSON is not valid")
	}
}

func TestDefaultAgentCard_Fields(t *testing.T) {
	card := DefaultAgentCard("http://localhost:8080")
	if card.Name == "" {
		t.Error("Name should not be empty")
	}
	if card.URL != "http://localhost:8080" {
		t.Errorf("URL = %q, want %q", card.URL, "http://localhost:8080")
	}
	if len(card.Capabilities) == 0 {
		t.Error("Capabilities should not be empty")
	}
	if len(card.Skills) == 0 {
		t.Error("Skills should not be empty")
	}
	if card.Authentication == nil {
		t.Error("Authentication should not be nil")
	}
	if len(card.DefaultInputModes) == 0 {
		t.Error("DefaultInputModes should not be empty")
	}
	if len(card.DefaultOutputModes) == 0 {
		t.Error("DefaultOutputModes should not be empty")
	}
}

func TestAgentCardJSON_Valid(t *testing.T) {
	jsonStr, err := AgentCardJSON("http://localhost:8080")
	if err != nil {
		t.Fatalf("AgentCardJSON: %v", err)
	}
	if !json.Valid([]byte(jsonStr)) {
		t.Fatal("AgentCardJSON output is not valid JSON")
	}
}

func TestDefaultAgentCard_Capabilities(t *testing.T) {
	card := DefaultAgentCard("http://localhost:8080")
	expected := []string{"negotiate", "query_price", "task", "mandate"}
	if len(card.Capabilities) != len(expected) {
		t.Fatalf("got %d capabilities, want %d", len(card.Capabilities), len(expected))
	}
	for i, cap := range card.Capabilities {
		if cap != expected[i] {
			t.Errorf("capabilities[%d] = %q, want %q", i, cap, expected[i])
		}
	}
}

func TestDefaultAgentCard_Skills(t *testing.T) {
	card := DefaultAgentCard("http://localhost:8080")
	if len(card.Skills) != 3 {
		t.Fatalf("got %d skills, want 3", len(card.Skills))
	}

	skillIDs := make(map[string]bool)
	for _, s := range card.Skills {
		if s.ID == "" {
			t.Error("skill ID should not be empty")
		}
		if s.Name == "" {
			t.Error("skill name should not be empty")
		}
		if s.Description == "" {
			t.Error("skill description should not be empty")
		}
		skillIDs[s.ID] = true
	}

	for _, id := range []string{"query_price", "negotiate", "mandate"} {
		if !skillIDs[id] {
			t.Errorf("missing skill: %s", id)
		}
	}
}

func TestStaticAgentCardJSON_Valid(t *testing.T) {
	jsonStr := StaticAgentCardJSON()
	if jsonStr == "" {
		t.Fatal("static agent card JSON is empty")
	}
	if !json.Valid([]byte(jsonStr)) {
		t.Fatal("static agent card JSON is not valid JSON")
	}
}

func TestStaticAgentCardJSON_Unmarshal(t *testing.T) {
	jsonStr := StaticAgentCardJSON()
	var card AgentCard
	if err := json.Unmarshal([]byte(jsonStr), &card); err != nil {
		t.Fatalf("unmarshal static agent card: %v", err)
	}
	if card.Name == "" {
		t.Error("Name should not be empty")
	}
	if len(card.Skills) != 3 {
		t.Errorf("got %d skills, want 3", len(card.Skills))
	}
}
