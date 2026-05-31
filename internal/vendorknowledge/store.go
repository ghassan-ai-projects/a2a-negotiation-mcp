package vendorknowledge

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages vendor knowledge documents in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a VendorKnowledgeStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate vendorknowledge: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vendor_knowledge_docs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		content TEXT NOT NULL,
		doc_type TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// IngestDocument saves a new vendor knowledge document and returns it with the generated ID and timestamp.
func (s *Store) IngestDocument(ctx context.Context, vendor, content, docType string) (*KnowledgeDoc, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_knowledge_docs (vendor, content, doc_type, created_at)
		VALUES (?, ?, ?, ?)
	`, vendor, content, docType, now)
	if err != nil {
		return nil, fmt.Errorf("ingest document: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("ingest document last insert id: %w", err)
	}
	return &KnowledgeDoc{
		ID:        int(id),
		Vendor:    vendor,
		Content:   content,
		DocType:   docType,
		CreatedAt: now,
	}, nil
}

// SearchDocs searches vendor knowledge documents by vendor and content query.
// Results are ordered by created_at DESC.
func (s *Store) SearchDocs(ctx context.Context, vendor, query string) ([]KnowledgeDoc, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, content, doc_type, created_at
		FROM vendor_knowledge_docs
		WHERE vendor = ? AND content LIKE '%' || ? || '%'
		ORDER BY created_at DESC
	`, vendor, query)
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	defer rows.Close()

	var docs []KnowledgeDoc
	for rows.Next() {
		var d KnowledgeDoc
		if err := rows.Scan(&d.ID, &d.Vendor, &d.Content, &d.DocType, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan doc: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if docs == nil {
		docs = []KnowledgeDoc{}
	}
	return docs, nil
}

// GetKnowledgeReport returns a summary report for a vendor's knowledge documents.
func (s *Store) GetKnowledgeReport(ctx context.Context, vendor string) (map[string]any, error) {
	// Total docs count
	var totalDocs int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vendor_knowledge_docs WHERE vendor = ?
	`, vendor).Scan(&totalDocs)
	if err != nil {
		return nil, fmt.Errorf("knowledge report count: %w", err)
	}

	// Doc type breakdown
	typeRows, err := s.db.QueryContext(ctx, `
		SELECT doc_type, COUNT(*) as cnt
		FROM vendor_knowledge_docs
		WHERE vendor = ?
		GROUP BY doc_type
		ORDER BY cnt DESC
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("knowledge report type breakdown: %w", err)
	}
	defer typeRows.Close()

	docTypeBreakdown := make(map[string]int)
	for typeRows.Next() {
		var docType string
		var count int
		if err := typeRows.Scan(&docType, &count); err != nil {
			return nil, fmt.Errorf("scan type breakdown: %w", err)
		}
		docTypeBreakdown[docType] = count
	}
	if err := typeRows.Err(); err != nil {
		return nil, err
	}

	// Most recent document
	var mostRecent *string
	var recentCreatedAt string
	err = s.db.QueryRowContext(ctx, `
		SELECT created_at FROM vendor_knowledge_docs
		WHERE vendor = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, vendor).Scan(&recentCreatedAt)
	if err == sql.ErrNoRows {
		mostRecent = nil
	} else if err != nil {
		return nil, fmt.Errorf("knowledge report most recent: %w", err)
	} else {
		mostRecent = &recentCreatedAt
	}

	return map[string]any{
		"total_docs":         totalDocs,
		"doc_type_breakdown": docTypeBreakdown,
		"most_recent":        mostRecent,
	}, nil
}
