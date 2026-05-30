package a2a

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// AgentCard describes the agent's capabilities according to the A2A Agent Card spec.
type AgentCard struct {
	Schema             string           `json:"$schema,omitempty"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	URL                string           `json:"url"`
	Version            string           `json:"version"`
	Capabilities       []string         `json:"capabilities"`
	Skills             []Skill          `json:"skills"`
	Authentication     *Authentication  `json:"authentication,omitempty"`
	DefaultInputModes  []string         `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string         `json:"defaultOutputModes,omitempty"`
}

// Skill describes a specific capability of the agent.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URI         string   `json:"uri,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// Authentication describes authentication requirements.
type Authentication struct {
	Schemes []string `json:"schemes"`
	Bearer  bool     `json:"bearer,omitempty"`
}

// DefaultAgentCard returns the default Agent Card for this server.
func DefaultAgentCard(baseURL string) *AgentCard {
	return &AgentCard{
		Schema:      "https://a2a-api.agency/v1",
		Name:        "A2A Negotiation MCP Server",
		Description: "Agent-to-Agent SaaS pricing negotiation server. Query market prices, run multi-round negotiations across 3 strategy profiles, and manage pricing mandates.",
		URL:         baseURL,
		Version:     "1.0.0",
		Capabilities: []string{
			"negotiate",
			"query_price",
			"task",
			"mandate",
		},
		Skills: []Skill{
			{
				ID:          "query_price",
				Name:        "Query Market Price",
				Description: "Query fair market price range for a SaaS vendor's product. Supports quantity and contract-term multipliers.",
				URI:         baseURL + "/a2a/task",
				Examples:    []string{`{"task":"query_price","params":{"vendor":"Slack","sku":"Pro","quantity":500}}`},
			},
			{
				ID:          "negotiate",
				Name:        "Negotiate SaaS Pricing",
				Description: "Run a multi-round negotiation for a vendor/SKU using aggressive, balanced, or conservative strategy. Creates a mandate and a negotiation session.",
				URI:         baseURL + "/a2a/negotiate",
				Examples:    []string{`{"vendor":"Slack","sku":"Pro","strategy":"balanced","budget":7.50}`},
			},
			{
				ID:          "mandate",
				Name:        "Manage Pricing Mandates",
				Description: "Create and manage AP2-style mandates for intent, cart, and payment operations. Supports settlement, cancellation, and expiry.",
				URI:         baseURL + "/a2a/task",
				Examples:    []string{`{"task":"mandate_create","params":{"type":"intent","principal":"agent-alpha","terms":{"vendor":"Slack"}}}`},
			},
		},
		Authentication: &Authentication{
			Schemes: []string{"none"},
			Bearer:  false,
		},
		DefaultInputModes:  []string{"text/json"},
		DefaultOutputModes: []string{"text/json"},
	}
}

// AgentCardJSON returns the agent card as a JSON string.
func AgentCardJSON(baseURL string) (string, error) {
	card := DefaultAgentCard(baseURL)
	b, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal agent card: %w", err)
	}
	return string(b), nil
}

//go:embed agent_card_static.json
var staticAgentCardJSON string

// StaticAgentCardJSON returns the embedded static agent card JSON.
func StaticAgentCardJSON() string {
	return staticAgentCardJSON
}
