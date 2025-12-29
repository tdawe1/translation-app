package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// LemonSqueezyWebhookEvent represents a LemonSqueezy webhook payload
type LemonSqueezyWebhookEvent struct {
	Meta struct {
		EventID    string `json:"event_id"`
		CustomData map[string]string `json:"custom_data,omitempty"`
		TestMode   bool   `json:"test_mode"`
	} `json:"meta"`
	Data struct {
		Attributes struct {
			FirstSubscriptionItem struct {
				ID string `json:"id"`
			} `json:"first_subscription_item"`
			Status           string  `json:"status"`
			StatusFormatted  string  `json:"status_formatted"`
			TrialEndsAt      *string `json:"trial_ends_at,omitempty"`
			RenewsAt         *string `json:"renews_at,omitempty"`
			EndsAt           *string `json:"ends_at,omitempty"`
			UpdatePaymentMethodURL *string `json:"update_payment_method_url"`
			CustomerID       string `json:"customer_id"`
			ProductID        string `json:"product_id"`
			VariantID        string `json:"variant_id"`
			UserID           string `json:"user_id"`
		} `json:"attributes"`
	} `json:"data"`
}

// LemonSqueezyHandler handles LemonSqueezy webhooks
type LemonSqueezyHandler struct {
	webhookSecret string
	db            database.Database
}

// NewLemonSqueezyHandler creates a new LemonSqueezy webhook handler
func NewLemonSqueezyHandler(webhookSecret string, db database.Database) *LemonSqueezyHandler {
	return &LemonSqueezyHandler{
		webhookSecret: webhookSecret,
		db:            db,
	}
}

// HandleWebhook processes incoming LemonSqueezy webhooks
func (h *LemonSqueezyHandler) HandleWebhook(c *fiber.Ctx) error {
	// Get signature from headers
	signature := c.Get("X-Signature")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing signature",
			"code":  "MISSING_SIGNATURE",
		})
	}

	// Read body
	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Empty body",
			"code":  "EMPTY_BODY",
		})
	}

	// Verify signature
	if !h.verifySignature(body, signature) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature",
			"code":  "INVALID_SIGNATURE",
		})
	}

	// Parse event
	var event LemonSqueezyWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse event",
			"code":  "PARSE_ERROR",
		})
	}

	// Check for duplicate event (idempotency)
	var existingEvent models.BillingEvent
	result := h.db.Where("event_id = ?", event.Meta.EventID).First(&existingEvent)
	if result.Error == nil {
		// Already processed, return success
		return c.SendStatus(fiber.StatusOK)
	}

	// Process event based on type
	if err := h.processEvent(&event); err != nil {
		log.Printf("Failed to process event %s: %v", event.Meta.EventID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process event",
			"code":  "PROCESS_ERROR",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

// verifySignature verifies the webhook signature
func (h *LemonSqueezyHandler) verifySignature(body []byte, signature string) bool {
	// Signature format: <timestamp>.<hex_signature>
	// For now, just verify the hex part
	// In production, you'd want to verify timestamp too

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures securely
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// processEvent processes a webhook event
func (h *LemonSqueezyHandler) processEvent(event *LemonSqueezyWebhookEvent) error {
	// Store event for idempotency
	eventData, _ := json.Marshal(event.Data)
	billingEvent := models.BillingEvent{
		EventID:   event.Meta.EventID,
		EventType: h.getEventType(event),
		EventData: string(eventData),
	}

	// Extract user_id from custom data or attributes
	var userID *uuid.UUID
	if userIDStr, ok := event.Meta.CustomData["user_id"]; ok {
		if parsedUUID, err := uuid.Parse(userIDStr); err == nil {
			userID = &parsedUUID
			billingEvent.UserID = userID
		}
	}

	// Process based on event type
	switch h.getEventType(event) {
	case "subscription_created", "subscription_updated", "subscription_resumed":
		return h.handleSubscriptionActive(event, userID)
	case "subscription_cancelled":
		return h.handleSubscriptionCancelled(event, userID)
	case "subscription_paused":
		return h.handleSubscriptionPaused(event, userID)
	case "subscription_unpaused":
		return h.handleSubscriptionUnpaused(event, userID)
	case "subscription_expired":
		return h.handleSubscriptionExpired(event, userID)
	}

	// Save billing event
	return h.db.Create(&billingEvent).Error
}

// getEventType extracts the event type from the event
func (h *LemonSqueezyHandler) getEventType(event *LemonSqueezyWebhookEvent) string {
	// LemonSqueezy doesn't always send explicit event_type in payload
	// We can infer from status and attributes
	status := event.Data.Attributes.Status

	switch status {
	case "active":
		return "subscription_created"
	case "cancelled":
		return "subscription_cancelled"
	case "paused":
		return "subscription_paused"
	case "unpaused":
		return "subscription_unpaused"
	case "expired":
		return "subscription_expired"
	case "past_due":
		return "subscription_past_due"
	default:
		return fmt.Sprintf("subscription_%s", status)
	}
}

// handleSubscriptionActive handles subscription activation/updates
func (h *LemonSqueezyHandler) handleSubscriptionActive(event *LemonSqueezyWebhookEvent, userID *uuid.UUID) error {
	if userID == nil {
		return fmt.Errorf("no user_id in event")
	}

	// Use the subscription item ID from LemonSqueezy
	lemonSubscriptionID := event.Data.Attributes.FirstSubscriptionItem.ID

	// Find or create subscription
	var subscription models.Subscription
	result := h.db.Where("lemon_subscription_id = ?", lemonSubscriptionID).First(&subscription)

	if result.Error != nil {
		// Create new subscription
		subscription = models.Subscription{
			UserID:             *userID,
			LemonSubscriptionID: lemonSubscriptionID,
			SubscriptionStatus:  event.Data.Attributes.Status,
		}
	} else {
		// Update existing
		subscription.SubscriptionStatus = event.Data.Attributes.Status
	}

	return h.db.Save(&subscription).Error
}

// handleSubscriptionCancelled handles subscription cancellation
func (h *LemonSqueezyHandler) handleSubscriptionCancelled(event *LemonSqueezyWebhookEvent, userID *uuid.UUID) error {
	if userID == nil {
		return fmt.Errorf("no user_id in event")
	}

	return h.db.Model(&models.Subscription{}).
		Where("user_id = ?", *userID).
		Update("subscription_status", "cancelled").
		Error
}

// handleSubscriptionPaused handles subscription pause
func (h *LemonSqueezyHandler) handleSubscriptionPaused(event *LemonSqueezyWebhookEvent, userID *uuid.UUID) error {
	if userID == nil {
		return fmt.Errorf("no user_id in event")
	}

	return h.db.Model(&models.Subscription{}).
		Where("user_id = ?", *userID).
		Update("subscription_status", "paused").
		Error
}

// handleSubscriptionUnpaused handles subscription resume
func (h *LemonSqueezyHandler) handleSubscriptionUnpaused(event *LemonSqueezyWebhookEvent, userID *uuid.UUID) error {
	if userID == nil {
		return fmt.Errorf("no user_id in event")
	}

	return h.db.Model(&models.Subscription{}).
		Where("user_id = ?", *userID).
		Update("subscription_status", "active").
		Error
}

// handleSubscriptionExpired handles subscription expiry
func (h *LemonSqueezyHandler) handleSubscriptionExpired(event *LemonSqueezyWebhookEvent, userID *uuid.UUID) error {
	if userID == nil {
		return fmt.Errorf("no user_id in event")
	}

	return h.db.Model(&models.Subscription{}).
		Where("user_id = ?", *userID).
		Update("subscription_status", "expired").
		Error
}
