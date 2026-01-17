package models

import (
	"time"

	"github.com/google/uuid"
)

// TranslationJobStatus represents the status of a translation job
type TranslationJobStatus string

const (
	TranslationJobStatusPending         TranslationJobStatus = "pending"
	TranslationJobStatusProcessing      TranslationJobStatus = "processing"
	TranslationJobStatusTranslating     TranslationJobStatus = "translating"
	TranslationJobStatusPendingApproval TranslationJobStatus = "pending_approval"
	TranslationJobStatusApproved        TranslationJobStatus = "approved"
	TranslationJobStatusRejected        TranslationJobStatus = "rejected"
	TranslationJobStatusCompleted       TranslationJobStatus = "completed"
	TranslationJobStatusFailed          TranslationJobStatus = "failed"
	TranslationJobStatusCancelled       TranslationJobStatus = "cancelled"
)

// TranslationJob represents a translation job in the system
// This model stores job metadata in PostgreSQL while the Python worker
// manages the actual translation state in Redis
type TranslationJob struct {
	Base
	UserID           uuid.UUID            `gorm:"type:uuid;not null;index" json:"user_id"`
	SourceFile       string               `gorm:"size:500;not null" json:"source_file"`
	TargetFile       string               `gorm:"size:500" json:"target_file,omitempty"`
	SourceLang       string               `gorm:"size:10;default:'ja'" json:"source_lang"`
	TargetLang       string               `gorm:"size:10;default:'en'" json:"target_lang"`
	Status           TranslationJobStatus `gorm:"size:30;default:'pending';index" json:"status"`
	ProjectType      string               `gorm:"size:30;default:'routine'" json:"project_type"` // "critical" or "routine"
	ApprovalMode     string               `gorm:"size:30;default:'async'" json:"approval_mode"`  // "blocking" or "async"
	OverallScore     float64              `gorm:"default:0" json:"overall_score"`
	SegmentCount     int                  `gorm:"default:0" json:"segment_count"`
	FlaggedCount     int                  `gorm:"default:0" json:"flagged_count"`
	JudgeResolutions int                  `gorm:"default:0" json:"judge_resolutions"`
	Progress         float64              `gorm:"default:0" json:"progress"` // 0.0 to 1.0
	Error            string               `gorm:"type:text" json:"error,omitempty"`
	WorkerID         string               `gorm:"size:100" json:"worker_id,omitempty"`
	RedisJobID       string               `gorm:"size:100;index" json:"redis_job_id,omitempty"`
	HasUserEdits     bool                 `gorm:"default:false;index" json:"has_user_edits"`
	CompletedAt      *time.Time           `json:"completed_at,omitempty"`
	ApprovedAt       *time.Time           `json:"approved_at,omitempty"`
	ApprovedBy       string               `gorm:"size:255" json:"approved_by,omitempty"`

	// Relationships
	User     User                 `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Segments []TranslationSegment `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"segments,omitempty"`
}

// TableName returns the table name for GORM
func (TranslationJob) TableName() string {
	return "translation_jobs"
}

// IsTerminal returns true if the job is in a terminal state
func (j *TranslationJob) IsTerminal() bool {
	switch j.Status {
	case TranslationJobStatusApproved, TranslationJobStatusRejected,
		TranslationJobStatusCompleted, TranslationJobStatusFailed,
		TranslationJobStatusCancelled:
		return true
	}
	return false
}

// CanApprove returns true if the job can be approved
func (j *TranslationJob) CanApprove() bool {
	return j.Status == TranslationJobStatusPendingApproval
}

// CanAutoApprove checks if the job meets auto-approval threshold
func (j *TranslationJob) CanAutoApprove(threshold float64) bool {
	if j.Status != TranslationJobStatusPendingApproval {
		return false
	}
	return j.OverallScore >= threshold && j.FlaggedCount == 0
}

// TranslationSegment represents a single translatable segment within a job
type TranslationSegment struct {
	Base
	JobID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_segment_job_segment,priority:1" json:"job_id"`
	SegmentID       string     `gorm:"size:100;not null;index:idx_segment_job_segment,priority:2" json:"segment_id"` // e.g., "slide1_title", "page_3"
	Source          string     `gorm:"type:text;not null" json:"source"`
	Target          string     `gorm:"type:text" json:"target"`
	Context         string     `gorm:"type:jsonb;default:'{}'" json:"context"` // JSON context (slide number, element type, etc.)
	JudgeWinner     string     `gorm:"size:30" json:"judge_winner,omitempty"`  // "model_a", "model_b", "edited", "tie"
	JudgeConfidence float64    `gorm:"default:0" json:"judge_confidence"`
	JudgeReasoning  string     `gorm:"type:text" json:"judge_reasoning,omitempty"`
	IsFlagged       bool       `gorm:"default:false;index" json:"is_flagged"`
	FlagReason      string     `gorm:"type:text" json:"flag_reason,omitempty"`
	ModelAOutput    string     `gorm:"type:text" json:"model_a_output,omitempty"`
	ModelBOutput    string     `gorm:"type:text" json:"model_b_output,omitempty"`
	GlossaryTerms   string     `gorm:"type:jsonb;default:'[]'" json:"glossary_terms"` // JSON array of matched terms
	EditedBy        string     `gorm:"size:255" json:"edited_by,omitempty"`
	EditedAt        *time.Time `json:"edited_at,omitempty"`

	// Relationship
	Job TranslationJob `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName returns the table name for GORM
func (TranslationSegment) TableName() string {
	return "translation_segments"
}

// IsEdited returns true if the segment was manually edited
func (s *TranslationSegment) IsEdited() bool {
	return s.JudgeWinner == "edited" || s.EditedBy != ""
}

// NeedsReview returns true if the segment needs human review
func (s *TranslationSegment) NeedsReview() bool {
	return s.IsFlagged || s.JudgeConfidence < 0.7
}
