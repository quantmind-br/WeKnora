// Package types defines data structures and types used throughout the system
// These types are shared across different service modules to ensure data consistency
package types

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// ChunkType defines the different types of Chunk
type ChunkType = string

const (
	// ChunkTypeText represents a regular text Chunk
	ChunkTypeText ChunkType = "text"
	// ChunkTypeParentText represents the parent text Chunk in the parent-child chunking strategy (used for context only, not included in vector indexing)
	ChunkTypeParentText ChunkType = "parent_text"
	// ChunkTypeImageOCR represents a Chunk of image OCR text
	ChunkTypeImageOCR ChunkType = "image_ocr"
	// ChunkTypeImageCaption represents a Chunk of image caption
	ChunkTypeImageCaption ChunkType = "image_caption"
	// ChunkTypeSummary represents a summary-type Chunk
	ChunkTypeSummary = "summary"
	// ChunkTypeEntity represents an entity-type Chunk
	ChunkTypeEntity ChunkType = "entity"
	// ChunkTypeRelationship represents a relationship-type Chunk
	ChunkTypeRelationship ChunkType = "relationship"
	// ChunkTypeFAQ represents an FAQ entry Chunk
	ChunkTypeFAQ ChunkType = "faq"
	// ChunkTypeWebSearch represents a Chunk of web search results
	ChunkTypeWebSearch ChunkType = "web_search"
	// ChunkTypeTableSummary represents a Chunk of data table summary
	ChunkTypeTableSummary ChunkType = "table_summary"
	// ChunkTypeTableColumn represents a Chunk of data table column description
	ChunkTypeTableColumn ChunkType = "table_column"
	// ChunkTypeWikiPage represents a Chunk synced from a wiki page, used to integrate wiki pages into the existing retrieval pipeline
	ChunkTypeWikiPage ChunkType = "wiki_page"
)

// ChunkStatus defines the different statuses of a Chunk
type ChunkStatus int

const (
	ChunkStatusDefault ChunkStatus = 0
	// ChunkStatusStored represents a Chunk that has been stored
	ChunkStatusStored ChunkStatus = 1
	// ChunkStatusIndexed represents a Chunk that has been indexed
	ChunkStatusIndexed ChunkStatus = 2
)

// ChunkFlags defines the flag bits for a Chunk, used to manage multiple boolean states
type ChunkFlags int

const (
	// ChunkFlagRecommended represents the recommendable state (1 << 0 = 1)
	// When this flag is set, the Chunk can be recommended to users
	ChunkFlagRecommended ChunkFlags = 1 << 0
	// Can be extended with more flags in the future:
	// ChunkFlagPinned ChunkFlags = 1 << 1  // pinned
	// ChunkFlagHot    ChunkFlags = 1 << 2  // trending
)

// HasFlag checks whether the specified flag is set
func (f ChunkFlags) HasFlag(flag ChunkFlags) bool {
	return f&flag != 0
}

// SetFlag sets the specified flag
func (f ChunkFlags) SetFlag(flag ChunkFlags) ChunkFlags {
	return f | flag
}

// ClearFlag clears the specified flag
func (f ChunkFlags) ClearFlag(flag ChunkFlags) ChunkFlags {
	return f &^ flag
}

// ToggleFlag toggles the specified flag
func (f ChunkFlags) ToggleFlag(flag ChunkFlags) ChunkFlags {
	return f ^ flag
}

// ImageInfo represents image information associated with a Chunk
type ImageInfo struct {
	// Image URL (COS)
	URL string `json:"url"          gorm:"type:text"`
	// Original image URL
	OriginalURL string `json:"original_url" gorm:"type:text"`
	// Start position of the image in the text
	StartPos int `json:"start_pos"`
	// End position of the image in the text
	EndPos int `json:"end_pos"`
	// Image description
	Caption string `json:"caption"`
	// Image OCR text
	OCRText string `json:"ocr_text"`
}

// VideoInfo represents video information associated with a Chunk
type VideoInfo struct {
	// VideoURL
	URL string `json:"url"          gorm:"type:text"`
}

// Chunk represents a document chunk
// Chunks are meaningful text segments extracted from original documents
// and are the basic units of knowledge base retrieval
// Each chunk contains a portion of the original content
// and maintains its positional relationship with the original text
// Chunks can be independently embedded as vectors and retrieved, supporting precise content localization
type Chunk struct {
	// Unique identifier of the chunk, using UUID format
	ID string `json:"id"                       gorm:"type:varchar(36);primaryKey"`
	// SeqID is an auto-increment integer ID for external API usage (FAQ entries)
	SeqID int64 `json:"seq_id"                   gorm:"type:bigint;uniqueIndex;autoIncrement"`
	// Tenant ID, used for multi-tenant isolation
	TenantID uint64 `json:"tenant_id"`
	// ID of the parent knowledge, associated with the Knowledge model
	KnowledgeID string `json:"knowledge_id"`
	// ID of the knowledge base, for quick location
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// Optional tag ID for categorization within a knowledge base (used for FAQ)
	TagID string `json:"tag_id"                   gorm:"type:varchar(36);index"`
	// Actual text content of the chunk
	Content string `json:"content"`
	// SourceContent is the immutable parser output. Legacy rows are lazily
	// backfilled from Content on the first manual edit.
	SourceContent string `json:"-"`
	// ContentRevision is incremented for every user edit or rollback.
	ContentRevision int `json:"content_revision" gorm:"not null;default:0"`
	// IndexStatus reports whether the current content is reflected in the
	// retrieval stores: ready | processing | failed.
	IndexStatus string `json:"index_status" gorm:"type:varchar(16);not null;default:'ready'"`
	// LastEditorID records the actor that produced the current revision.
	LastEditorID string `json:"last_editor_id" gorm:"type:varchar(64);not null;default:''"`
	// Index position of the chunk in the original document
	ChunkIndex int `json:"chunk_index"`
	// Whether the chunk is enabled, can be used to temporarily disable certain chunks
	IsEnabled bool `json:"is_enabled"               gorm:"default:true"`
	// Flags stores multiple boolean states as bit flags (e.g. recommended status)
	// Default value is ChunkFlagRecommended (1), meaning recommendable by default
	Flags ChunkFlags `json:"flags"                    gorm:"default:1"`
	// Status of the chunk
	Status int `json:"status"                   gorm:"default:0"`
	// Starting character position in the original text
	StartAt int `json:"start_at"`
	// Ending character position in the original text
	EndAt int `json:"end_at"`
	// Previous chunk ID
	PreChunkID string `json:"pre_chunk_id"`
	// Next chunk ID
	NextChunkID string `json:"next_chunk_id"`
	// Chunk type, distinguishes different kinds of chunks
	ChunkType ChunkType `json:"chunk_type"               gorm:"type:varchar(20);default:'text'"`
	// Parent Chunk ID, links image chunks to the original text chunk
	ParentChunkID string `json:"parent_chunk_id"          gorm:"type:varchar(36);index"`
	// Relation Chunk ID, used to link the relation Chunk with the original text Chunk
	RelationChunks JSON `json:"relation_chunks"          gorm:"type:json"`
	// Indirect relation Chunk ID, used to link the indirect relation Chunk with the original text Chunk
	IndirectRelationChunks JSON `json:"indirect_relation_chunks" gorm:"type:json"`
	// Metadata stores chunk-level extended information, such as FAQ metadata
	Metadata JSON `json:"metadata"                 gorm:"type:json"`
	// ContentHash stores the content's hash value for fast matching (mainly used for FAQ)
	ContentHash string `json:"content_hash"             gorm:"type:varchar(64)"`
	// Image information, stored as JSON
	ImageInfo string `json:"image_info"               gorm:"type:text"`
	// Chunk creation time
	CreatedAt time.Time `json:"created_at"`
	// Chunk last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete marker, supports data recovery
	DeletedAt gorm.DeletedAt `json:"deleted_at"               gorm:"index"`
	// ContextHeader is a Markdown heading breadcrumb prepended when indexing.
	// It is persisted so a later content edit can rebuild the same index input.
	ContextHeader string `json:"-" gorm:"type:text"`
}

// ChunkRevision is an immutable snapshot of a superseded chunk revision.
// The current content lives on Chunk; this table stores prior versions.
type ChunkRevision struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id" gorm:"index"`
	KnowledgeBaseID string    `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	KnowledgeID     string    `json:"knowledge_id" gorm:"type:varchar(36);index"`
	ChunkID         string    `json:"chunk_id" gorm:"type:varchar(36);uniqueIndex:idx_chunk_revision"`
	Revision        int       `json:"revision" gorm:"uniqueIndex:idx_chunk_revision"`
	Content         string    `json:"content" gorm:"type:text"`
	IsEnabled       bool      `json:"is_enabled"`
	EditorID        string    `json:"editor_id" gorm:"type:varchar(64)"`
	EditSource      string    `json:"edit_source" gorm:"type:varchar(16)"`
	EditedAt        time.Time `json:"edited_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// EmbeddingContent returns the chunk content with ContextHeader prepended
// when set. Use this where the embedding model needs section context that
// isn't part of the literal Content. Surrounding whitespace on Content is
// trimmed so leading/trailing newlines from boundary slicing don't dilute
// the embedded vector.
func (c *Chunk) EmbeddingContent() string {
	if c == nil {
		return ""
	}
	body := strings.TrimSpace(c.Content)
	if c.ContextHeader == "" {
		return body
	}
	return c.ContextHeader + "\n\n" + body
}

// AssignChunkSeqIDs assigns sequential SeqIDs to a batch of chunks that have SeqID == 0.
// Must be called before CreateInBatches for SQLite compatibility.
func AssignChunkSeqIDs(tx *gorm.DB, chunks []*Chunk) error {
	needAssign := false
	for _, c := range chunks {
		if c.SeqID == 0 {
			needAssign = true
			break
		}
	}
	if !needAssign {
		return nil
	}

	var maxSeqID *int64
	if err := tx.Unscoped().Model(&Chunk{}).Select("MAX(seq_id)").Scan(&maxSeqID).Error; err != nil {
		return err
	}
	next := int64(1)
	if maxSeqID != nil {
		next = *maxSeqID + 1
	}
	for _, c := range chunks {
		if c.SeqID == 0 {
			c.SeqID = next
			next++
		}
	}
	return nil
}
