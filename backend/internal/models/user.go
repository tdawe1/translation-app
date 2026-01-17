package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base contains common columns for all tables
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// BeforeCreate - GORM hook
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User represents a user account
type User struct {
	Base
	Email         string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	EmailVerified bool   `gorm:"default:false" json:"email_verified"`
	PasswordHash  string `gorm:"size:255" json:"-"` // Don't serialize in JSON
	IsActive      bool   `gorm:"default:true" json:"is_active"`
	Role          string `gorm:"size:20;default:'user';index" json:"role"` // 'admin', 'user'

	// OAuth provider (simplified - no token storage)
	// Provider: "google", "github", or empty for email/password
	Provider   string `gorm:"size:20;index:idx_provider,priority:1" json:"provider,omitempty"`
	ProviderID string `gorm:"size:255;index:idx_provider,priority:2" json:"provider_id,omitempty"`

	// Relationships
	OAuthAccounts []OAuthAccount `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"oauth_accounts,omitempty"`
	APIKeys       []APIKey       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"api_keys,omitempty"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"refresh_tokens,omitempty"`
	WatcherConfig *WatcherConfig `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"watcher_config,omitempty"`
	WatcherState  *WatcherState  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"watcher_state,omitempty"`
	Subscription  *Subscription  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"subscription,omitempty"`
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// OAuthAccount represents a linked OAuth provider account
type OAuthAccount struct {
	Base
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider       string     `gorm:"size:50;not null" json:"provider"` // 'google', 'github'
	ProviderUserID string     `gorm:"size:255;not null" json:"provider_user_id"`
	AccessToken    string     `gorm:"type:text" json:"access_token,omitempty"`
	RefreshToken   string     `gorm:"type:text" json:"refresh_token,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// APIKey represents a programmatic API key
type APIKey struct {
	Base
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	KeyHash   string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	KeyPrefix string     `gorm:"size:10;not null" json:"key_prefix"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	Scopes    string     `gorm:"type:jsonb" json:"scopes"` // JSON array
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

// RefreshToken represents a refresh token for sessions
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string     `gorm:"size:255;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// WatcherConfig represents per-user watcher configuration
type WatcherConfig struct {
	Base
	UserID                uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	RSSFeedURL            string    `gorm:"type:text;default:'https://gengo.com/jobs/rss'" json:"rss_feed_url"`
	WebSocketEnabled      bool      `gorm:"default:true" json:"websocket_enabled"`
	GengoUserID           string    `gorm:"size:50" json:"gengo_user_id,omitempty"`
	GengoUserKey          string    `gorm:"type:text" json:"gengo_user_key,omitempty"` // Browser localStorage userKey for WebSocket auth
	GengoSessionToken     string    `gorm:"type:text" json:"-"`                        // Encrypted, don't serialize
	MinReward             float64   `gorm:"default:0" json:"min_reward"`
	MaxReward             float64   `gorm:"default:999999" json:"max_reward"`
	IncludedLanguagePairs string    `gorm:"type:jsonb" json:"included_language_pairs"` // JSON array
	EnableDesktopNotifs   bool      `gorm:"default:true" json:"enable_desktop_notifications"`
	EnableSoundNotifs     bool      `gorm:"default:true" json:"enable_sound_notifications"`
	EnableEmailNotifs     bool      `gorm:"default:false" json:"enable_email_notifications"`
	NotificationEmail     string    `gorm:"size:255" json:"notification_email,omitempty"`
	AutoAcceptEnabled     bool      `gorm:"default:false" json:"auto_accept_enabled"`
	AutoAcceptMinReward   *float64  `json:"auto_accept_min_reward,omitempty"`
	AutoAcceptMaxReward   *float64  `json:"auto_accept_max_reward,omitempty"`
}

// WatcherState represents per-user watcher runtime state
type WatcherState struct {
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	LastSeenJobIDs    string    `gorm:"type:jsonb" json:"last_seen_job_ids"` // JSON array
	LastSeenRSSLink   string    `gorm:"type:text" json:"last_seen_rss_link,omitempty"`
	TotalJobsFound    int       `gorm:"default:0" json:"total_jobs_found"`
	TotalJobsAccepted int       `gorm:"default:0" json:"total_jobs_accepted"`
	TotalEarnings     float64   `gorm:"default:0" json:"total_earnings"`
	WatcherStatus     string    `gorm:"size:20;default:'stopped'" json:"watcher_status"`
	LastActivity      time.Time `gorm:"default:now()" json:"last_activity"`
	RecentJobHistory  string    `gorm:"type:jsonb" json:"recent_job_history"` // JSON array
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// SubscriptionPlan represents subscription tiers
type SubscriptionPlan struct {
	Base
	Name          string `gorm:"size:50;uniqueIndex;not null" json:"name"` // 'free', 'pro', 'enterprise'
	PriceCents    int    `gorm:"default:0" json:"price_cents"`
	Interval      string `gorm:"size:20;default:'month'" json:"interval"`
	Features      string `gorm:"type:jsonb" json:"features"` // JSON object
	StripePriceID string `gorm:"size:100" json:"stripe_price_id,omitempty"`
	LemonPriceID  string `gorm:"size:100" json:"lemon_price_id,omitempty"`
	IsActive      bool   `gorm:"default:true" json:"is_active"`
}

// Subscription represents user subscription
type Subscription struct {
	Base
	UserID               uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	PlanID               uuid.UUID  `gorm:"type:uuid" json:"plan_id,omitempty"`
	StripeCustomerID     string     `gorm:"size:100" json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string     `gorm:"size:100;uniqueIndex" json:"stripe_subscription_id,omitempty"`
	LemonSubscriptionID  string     `gorm:"size:100;uniqueIndex" json:"lemon_subscription_id,omitempty"`
	SubscriptionStatus   string     `gorm:"size:50" json:"subscription_status,omitempty"`
	CurrentPeriodStart   *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool       `gorm:"default:false" json:"cancel_at_period_end"`
}

// BillingEvent represents Stripe/LemonSqueezy webhook events
type BillingEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	EventID     string     `gorm:"size:100;uniqueIndex;not null" json:"event_id"`
	EventType   string     `gorm:"size:50;not null" json:"event_type"`
	EventData   string     `gorm:"type:jsonb" json:"event_data"`
	ProcessedAt time.Time  `gorm:"default:now()" json:"processed_at"`
}

// AuditLog represents security/compliance audit trail
type AuditLog struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	EventType string     `gorm:"size:50;not null" json:"event_type"`
	EventData string     `gorm:"type:jsonb" json:"event_data"`
	IPAddress string     `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent string     `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time  `gorm:"default:now()" json:"created_at"`
}
