package models

import (
	"encoding/json"
	"time"
)

// PaymentTransaction stores checkout session state for polling and webhook reconciliation.
type PaymentTransaction struct {
	Base
	SessionID     string     `gorm:"size:255;uniqueIndex;not null" json:"session_id"`
	PlanID        string     `gorm:"size:50;not null" json:"plan_id"`
	PlanName      string     `gorm:"size:100;not null" json:"plan_name"`
	Amount        float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	Currency      string     `gorm:"size:10;not null" json:"currency"`
	Status        string     `gorm:"size:50;not null" json:"status"`
	PaymentStatus string     `gorm:"size:50;not null" json:"payment_status"`
	UserEmail     *string    `gorm:"size:255" json:"user_email,omitempty"`
	MetadataJSON  string     `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"-"`
	CheckoutURL   *string    `gorm:"type:text" json:"-"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

// TableName keeps the GORM table name aligned with the bridge-era schema.
func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

// MetadataMap returns the stored metadata as a plain Go map.
func (p *PaymentTransaction) MetadataMap() map[string]string {
	if p.MetadataJSON == "" {
		return map[string]string{}
	}

	metadata := map[string]string{}
	if err := json.Unmarshal([]byte(p.MetadataJSON), &metadata); err != nil {
		return map[string]string{}
	}
	return metadata
}

// SetMetadataMap serializes checkout metadata for storage.
func (p *PaymentTransaction) SetMetadataMap(metadata map[string]string) error {
	if len(metadata) == 0 {
		p.MetadataJSON = "{}"
		return nil
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	p.MetadataJSON = string(payload)
	return nil
}
