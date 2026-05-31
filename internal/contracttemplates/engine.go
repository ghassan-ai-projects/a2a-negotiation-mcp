package contracttemplates

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Engine manages contract template operations.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a contract templates engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// CreateTemplate creates a new contract template.
func (e *Engine) CreateTemplate(ctx context.Context, name, category, content string) (*ContractTemplate, error) {
	t := &ContractTemplate{
		ID:        uuid.New().String(),
		Name:      name,
		Category:  category,
		Content:   content,
		CreatedAt: "",
	}
	if err := e.store.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	// Re-read to get server-generated created_at
	saved, err := e.store.Get(ctx, t.ID)
	if err != nil {
		return nil, fmt.Errorf("read created template: %w", err)
	}
	return saved, nil
}

// ListTemplates returns templates, optionally filtered by category.
func (e *Engine) ListTemplates(ctx context.Context, category string) ([]ContractTemplate, error) {
	return e.store.List(ctx, category)
}

// GenerateContract renders a template with vendor name and custom parameters.
func (e *Engine) GenerateContract(ctx context.Context, templateID, vendor string, params map[string]string) (*GeneratedContract, error) {
	t, err := e.store.Get(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	if t == nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	content := t.Content
	varsUsed := []string{"vendor"}

	// Replace {{vendor}} with vendor name
	content = strings.ReplaceAll(content, "{{vendor}}", vendor)

	// Replace {{param_name}} with corresponding values
	for k, v := range params {
		re := regexp.MustCompile(`\{\{` + regexp.QuoteMeta(k) + `\}\}`)
		if re.MatchString(content) {
			varsUsed = append(varsUsed, k)
		}
		content = re.ReplaceAllString(content, v)
	}

	return &GeneratedContract{
		TemplateID:    templateID,
		VendorName:    vendor,
		Content:       content,
		VariablesUsed: varsUsed,
	}, nil
}
