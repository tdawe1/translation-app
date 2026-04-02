package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v85"
	stripeSession "github.com/stripe/stripe-go/v85/checkout/session"
	stripeWebhook "github.com/stripe/stripe-go/v85/webhook"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/models"
)

var (
	// ErrBillingGatewayUnavailable indicates that the checkout provider is not configured.
	ErrBillingGatewayUnavailable = errors.New("billing gateway unavailable")
	// ErrBillingWebhookUnavailable indicates that Stripe webhook verification is not configured.
	ErrBillingWebhookUnavailable = errors.New("billing webhook unavailable")
)

type billingPlan struct {
	ID          string
	Name        string
	PriceCents  int64
	Currency    string
	Interval    string
	Description string
	Features    []string
}

type billingPlanResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Amount        float64  `json:"amount"`
	AmountDisplay string   `json:"amount_display"`
	Currency      string   `json:"currency"`
	Interval      string   `json:"interval"`
	Description   string   `json:"description"`
	Features      []string `json:"features"`
}

type billingCheckoutRequest struct {
	PlanID    string  `json:"plan_id"`
	OriginURL string  `json:"origin_url"`
	UserEmail *string `json:"user_email"`
}

type billingCheckoutResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
}

type billingStatusResponse struct {
	SessionID     string                 `json:"session_id"`
	Status        string                 `json:"status"`
	PaymentStatus string                 `json:"payment_status"`
	AmountTotal   int64                  `json:"amount_total"`
	Currency      string                 `json:"currency"`
	Transaction   map[string]interface{} `json:"transaction,omitempty"`
	Detail        string                 `json:"detail,omitempty"`
}

type billingGatewayRequest struct {
	Plan       billingPlan
	SuccessURL string
	CancelURL  string
	UserEmail  string
	Metadata   map[string]string
}

type billingGatewaySession struct {
	ID            string
	URL           string
	Status        string
	PaymentStatus string
	AmountTotal   int64
	Currency      string
	Metadata      map[string]string
}

type billingWebhookEvent struct {
	EventID       string
	EventType     string
	SessionID     string
	Status        string
	PaymentStatus string
	AmountTotal   int64
	Currency      string
	Metadata      map[string]string
	Raw           string
}

type billingGateway interface {
	CreateCheckoutSession(ctx context.Context, request billingGatewayRequest) (*billingGatewaySession, error)
	GetCheckoutStatus(ctx context.Context, sessionID string) (*billingGatewaySession, error)
	ParseWebhook(payload []byte, signature string) (*billingWebhookEvent, error)
}

// BillingHandler serves pricing plans plus checkout and status polling.
type BillingHandler struct {
	db      database.Database
	gateway billingGateway
}

var defaultBillingPlans = []billingPlan{
	{
		ID:          "pro",
		Name:        "Pro",
		PriceCents:  2900,
		Currency:    "usd",
		Interval:    "month",
		Description: "For individual translators who need live watcher alerts and review tools.",
		Features: []string{
			"Realtime job watcher",
			"Translation review workspace",
			"Priority email support",
			"3 team members included",
		},
	},
	{
		ID:          "team",
		Name:        "Team",
		PriceCents:  7900,
		Currency:    "usd",
		Interval:    "month",
		Description: "For fast-moving teams coordinating multiple translators and reviewers.",
		Features: []string{
			"Everything in Pro",
			"Unlimited watcher presets",
			"Shared review visibility",
			"Priority launch concierge",
		},
	},
}

// NewBillingHandler builds the checkout handler using Stripe when configured.
func NewBillingHandler(db database.Database, cfg *config.Config) *BillingHandler {
	return &BillingHandler{
		db:      db,
		gateway: newBillingGateway(cfg),
	}
}

func newBillingGateway(cfg *config.Config) billingGateway {
	if cfg.StripeAPIKey == "" {
		if cfg.IsProduction() {
			return disabledBillingGateway{reason: "STRIPE_API_KEY is not configured"}
		}
		return newMockBillingGateway()
	}

	return stripeBillingGateway{
		apiKey:        cfg.StripeAPIKey,
		webhookSecret: cfg.StripeWebhookSecret,
	}
}

// GetPlans returns the public launch pricing plans.
func (h *BillingHandler) GetPlans(c *fiber.Ctx) error {
	plans := make([]billingPlanResponse, 0, len(defaultBillingPlans))
	for _, plan := range defaultBillingPlans {
		plans = append(plans, plan.response())
	}

	return c.JSON(fiber.Map{"plans": plans})
}

// CreateCheckout creates a Stripe Checkout Session or a development-safe mock.
func (h *BillingHandler) CreateCheckout(c *fiber.Ctx) error {
	var request billingCheckoutRequest
	if err := c.BodyParser(&request); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid billing request")
	}

	plan, ok := lookupBillingPlan(request.PlanID)
	if !ok {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid plan selected")
	}

	origin := normalizeOrigin(request.OriginURL, c)
	successURL := fmt.Sprintf("%s/pricing?session_id={CHECKOUT_SESSION_ID}", origin)
	cancelURL := fmt.Sprintf("%s/pricing", origin)
	userEmail := stringValue(request.UserEmail)
	metadata := map[string]string{
		"plan_id":    plan.ID,
		"plan_name":  plan.Name,
		"user_email": userEmail,
		"source":     "gengowatcher_pricing",
	}

	session, err := h.gateway.CreateCheckoutSession(c.UserContext(), billingGatewayRequest{
		Plan:       plan,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		UserEmail:  userEmail,
		Metadata:   metadata,
	})
	if err != nil {
		return h.respondForGatewayError(c, err, "Billing is not configured")
	}

	transaction := &models.PaymentTransaction{
		SessionID:     session.ID,
		PlanID:        plan.ID,
		PlanName:      plan.Name,
		Amount:        float64(plan.PriceCents) / 100,
		Currency:      session.Currency,
		Status:        session.Status,
		PaymentStatus: session.PaymentStatus,
		CheckoutURL:   optionalString(session.URL),
	}
	if request.UserEmail != nil && *request.UserEmail != "" {
		transaction.UserEmail = request.UserEmail
	}
	if err := transaction.SetMetadataMap(metadata); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to store billing metadata")
	}
	if err := h.upsertTransaction(transaction); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to store billing transaction")
	}

	return c.JSON(billingCheckoutResponse{
		URL:       session.URL,
		SessionID: session.ID,
	})
}

// GetStatus polls the backing checkout provider and falls back to stored transaction state.
func (h *BillingHandler) GetStatus(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	transaction, err := h.findTransaction(sessionID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to load billing transaction")
	}

	session, err := h.gateway.GetCheckoutStatus(c.UserContext(), sessionID)
	if err != nil {
		if transaction != nil {
			return c.JSON(statusResponseFromTransaction(transaction, "Status temporarily unavailable from Stripe; using stored transaction state."))
		}
		return h.respondForGatewayError(c, err, "Unable to fetch billing status")
	}

	if transaction == nil {
		plan, ok := lookupBillingPlan(session.Metadata["plan_id"])
		if !ok {
			plan = defaultBillingPlans[0]
		}
		transaction = &models.PaymentTransaction{
			SessionID: session.ID,
			PlanID:    plan.ID,
			PlanName:  plan.Name,
		}
	}

	applyGatewayState(transaction, session)
	if err := h.upsertTransaction(transaction); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to update billing transaction")
	}

	updated, err := h.findTransaction(sessionID)
	if err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to load updated billing transaction")
	}

	return c.JSON(statusResponseFromGateway(updated, session, ""))
}

// HandleStripeWebhook verifies and records Stripe webhook events.
func (h *BillingHandler) HandleStripeWebhook(c *fiber.Ctx) error {
	event, err := h.gateway.ParseWebhook(c.Body(), c.Get("Stripe-Signature"))
	if err != nil {
		if errors.Is(err, ErrBillingWebhookUnavailable) {
			return RespondWithError(c, fiber.StatusServiceUnavailable, apperrors.ErrConfigError, err.Error())
		}
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid Stripe webhook")
	}

	if event.EventID != "" {
		var existing models.BillingEvent
		result := h.db.Where("event_id = ?", event.EventID).First(&existing)
		switch {
		case result.Error == nil:
			return c.JSON(fiber.Map{
				"received":       true,
				"event_type":     event.EventType,
				"session_id":     event.SessionID,
				"payment_status": event.PaymentStatus,
			})
		case result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound):
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to check billing event state")
		}

		record := &models.BillingEvent{
			EventID:   event.EventID,
			EventType: event.EventType,
			EventData: event.Raw,
		}
		if err := h.db.Create(record).Error; err != nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to store billing webhook")
		}
	}

	transaction, err := h.findTransaction(event.SessionID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to load billing transaction")
	}

	if transaction == nil {
		plan, ok := lookupBillingPlan(event.Metadata["plan_id"])
		if !ok {
			plan = defaultBillingPlans[0]
		}
		transaction = &models.PaymentTransaction{
			SessionID: event.SessionID,
			PlanID:    plan.ID,
			PlanName:  plan.Name,
		}
	}

	applyGatewayState(transaction, &billingGatewaySession{
		ID:            event.SessionID,
		Status:        event.Status,
		PaymentStatus: event.PaymentStatus,
		AmountTotal:   event.AmountTotal,
		Currency:      event.Currency,
		Metadata:      event.Metadata,
	})

	if err := h.upsertTransaction(transaction); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to update billing transaction")
	}

	return c.JSON(fiber.Map{
		"received":       true,
		"event_type":     event.EventType,
		"session_id":     event.SessionID,
		"payment_status": event.PaymentStatus,
	})
}

func (h *BillingHandler) findTransaction(sessionID string) (*models.PaymentTransaction, error) {
	transaction := &models.PaymentTransaction{}
	result := h.db.Where("session_id = ?", sessionID).First(transaction)
	if result.Error != nil {
		return nil, result.Error
	}
	return transaction, nil
}

func (h *BillingHandler) upsertTransaction(next *models.PaymentTransaction) error {
	existing, err := h.findTransaction(next.SessionID)
	switch {
	case err == nil:
		existing.PlanID = next.PlanID
		existing.PlanName = next.PlanName
		existing.Amount = next.Amount
		existing.Currency = next.Currency
		existing.Status = next.Status
		existing.PaymentStatus = next.PaymentStatus
		existing.MetadataJSON = next.MetadataJSON
		if next.UserEmail != nil {
			existing.UserEmail = next.UserEmail
		}
		if next.CheckoutURL != nil && *next.CheckoutURL != "" {
			existing.CheckoutURL = next.CheckoutURL
		}
		if next.ProcessedAt != nil {
			existing.ProcessedAt = next.ProcessedAt
		} else if next.PaymentStatus == "paid" && existing.ProcessedAt == nil {
			now := time.Now().UTC()
			existing.ProcessedAt = &now
		}
		return h.db.Save(existing).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		if next.ProcessedAt == nil && next.PaymentStatus == "paid" {
			now := time.Now().UTC()
			next.ProcessedAt = &now
		}
		return h.db.Create(next).Error
	default:
		return err
	}
}

func (h *BillingHandler) respondForGatewayError(c *fiber.Ctx, err error, fallbackMessage string) error {
	switch {
	case errors.Is(err, ErrBillingGatewayUnavailable), errors.Is(err, ErrBillingWebhookUnavailable):
		return RespondWithError(c, fiber.StatusServiceUnavailable, apperrors.ErrConfigError, err.Error())
	case errors.Is(err, fiber.ErrNotFound):
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrInvalidRequest, fallbackMessage)
	default:
		return RespondWithError(c, fiber.StatusBadGateway, apperrors.ErrInternal, fallbackMessage)
	}
}

func lookupBillingPlan(planID string) (billingPlan, bool) {
	for _, plan := range defaultBillingPlans {
		if plan.ID == planID {
			return plan, true
		}
	}
	return billingPlan{}, false
}

func (p billingPlan) response() billingPlanResponse {
	return billingPlanResponse{
		ID:            p.ID,
		Name:          p.Name,
		Amount:        float64(p.PriceCents) / 100,
		AmountDisplay: fmt.Sprintf("$%.0f", float64(p.PriceCents)/100),
		Currency:      p.Currency,
		Interval:      p.Interval,
		Description:   p.Description,
		Features:      p.Features,
	}
}

func normalizeOrigin(originURL string, c *fiber.Ctx) string {
	parsed, err := url.Parse(originURL)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
		return strings.TrimRight(parsed.String(), "/")
	}
	return strings.TrimRight(c.BaseURL(), "/")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func applyGatewayState(transaction *models.PaymentTransaction, session *billingGatewaySession) {
	if transaction == nil || session == nil {
		return
	}

	metadata := session.Metadata
	if len(metadata) == 0 {
		metadata = transaction.MetadataMap()
	}

	if planID := metadata["plan_id"]; planID != "" {
		transaction.PlanID = planID
	}
	if planName := metadata["plan_name"]; planName != "" {
		transaction.PlanName = planName
	}
	if userEmail := metadata["user_email"]; userEmail != "" && transaction.UserEmail == nil {
		transaction.UserEmail = optionalString(userEmail)
	}

	if session.AmountTotal > 0 {
		transaction.Amount = float64(session.AmountTotal) / 100
	}
	if session.Currency != "" {
		transaction.Currency = session.Currency
	}
	if session.Status != "" {
		transaction.Status = session.Status
	}
	if session.PaymentStatus != "" {
		transaction.PaymentStatus = session.PaymentStatus
	}
	_ = transaction.SetMetadataMap(metadata)
	if session.PaymentStatus == "paid" && transaction.ProcessedAt == nil {
		now := time.Now().UTC()
		transaction.ProcessedAt = &now
	}
}

func statusResponseFromTransaction(transaction *models.PaymentTransaction, detail string) billingStatusResponse {
	return billingStatusResponse{
		SessionID:     transaction.SessionID,
		Status:        transaction.Status,
		PaymentStatus: transaction.PaymentStatus,
		AmountTotal:   int64(transaction.Amount * 100),
		Currency:      transaction.Currency,
		Transaction:   transactionToMap(transaction),
		Detail:        detail,
	}
}

func statusResponseFromGateway(transaction *models.PaymentTransaction, session *billingGatewaySession, detail string) billingStatusResponse {
	status := ""
	paymentStatus := ""
	amountTotal := int64(0)
	currency := ""

	if session != nil {
		status = session.Status
		paymentStatus = session.PaymentStatus
		amountTotal = session.AmountTotal
		currency = session.Currency
	}
	if transaction != nil {
		if status == "" {
			status = transaction.Status
		}
		if paymentStatus == "" {
			paymentStatus = transaction.PaymentStatus
		}
		if amountTotal == 0 {
			amountTotal = int64(transaction.Amount * 100)
		}
		if currency == "" {
			currency = transaction.Currency
		}
	}

	return billingStatusResponse{
		SessionID:     transaction.SessionID,
		Status:        status,
		PaymentStatus: paymentStatus,
		AmountTotal:   amountTotal,
		Currency:      currency,
		Transaction:   transactionToMap(transaction),
		Detail:        detail,
	}
}

func transactionToMap(transaction *models.PaymentTransaction) map[string]interface{} {
	if transaction == nil {
		return nil
	}

	payload := map[string]interface{}{
		"session_id":     transaction.SessionID,
		"plan_id":        transaction.PlanID,
		"plan_name":      transaction.PlanName,
		"amount":         transaction.Amount,
		"currency":       transaction.Currency,
		"status":         transaction.Status,
		"payment_status": transaction.PaymentStatus,
		"metadata":       transaction.MetadataMap(),
		"created_at":     transaction.CreatedAt,
		"updated_at":     transaction.UpdatedAt,
	}
	if transaction.UserEmail != nil {
		payload["user_email"] = *transaction.UserEmail
	}
	if transaction.ProcessedAt != nil {
		payload["processed_at"] = transaction.ProcessedAt
	}
	return payload
}

type disabledBillingGateway struct {
	reason string
}

func (g disabledBillingGateway) CreateCheckoutSession(context.Context, billingGatewayRequest) (*billingGatewaySession, error) {
	return nil, fmt.Errorf("%w: %s", ErrBillingGatewayUnavailable, g.reason)
}

func (g disabledBillingGateway) GetCheckoutStatus(context.Context, string) (*billingGatewaySession, error) {
	return nil, fmt.Errorf("%w: %s", ErrBillingGatewayUnavailable, g.reason)
}

func (g disabledBillingGateway) ParseWebhook([]byte, string) (*billingWebhookEvent, error) {
	return nil, fmt.Errorf("%w: %s", ErrBillingWebhookUnavailable, g.reason)
}

type mockBillingGateway struct {
	mu       sync.Mutex
	sessions map[string]*billingGatewaySession
}

func newMockBillingGateway() *mockBillingGateway {
	return &mockBillingGateway{
		sessions: map[string]*billingGatewaySession{},
	}
}

func (g *mockBillingGateway) CreateCheckoutSession(_ context.Context, request billingGatewayRequest) (*billingGatewaySession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	sessionID := "mock_cs_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	url := strings.Replace(request.SuccessURL, "{CHECKOUT_SESSION_ID}", sessionID, 1)
	session := &billingGatewaySession{
		ID:            sessionID,
		URL:           url,
		Status:        "open",
		PaymentStatus: "pending",
		AmountTotal:   request.Plan.PriceCents,
		Currency:      request.Plan.Currency,
		Metadata:      copyMetadata(request.Metadata),
	}
	g.sessions[sessionID] = session
	return cloneGatewaySession(session), nil
}

func (g *mockBillingGateway) GetCheckoutStatus(_ context.Context, sessionID string) (*billingGatewaySession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	session, ok := g.sessions[sessionID]
	if !ok {
		return nil, fiber.ErrNotFound
	}

	session.Status = "complete"
	session.PaymentStatus = "paid"
	return cloneGatewaySession(session), nil
}

func (g *mockBillingGateway) ParseWebhook([]byte, string) (*billingWebhookEvent, error) {
	return nil, ErrBillingWebhookUnavailable
}

type stripeBillingGateway struct {
	apiKey        string
	webhookSecret string
}

func (g stripeBillingGateway) CreateCheckoutSession(_ context.Context, request billingGatewayRequest) (*billingGatewaySession, error) {
	stripe.Key = g.apiKey

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(request.SuccessURL),
		CancelURL:  stripe.String(request.CancelURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(request.Plan.Currency),
					UnitAmount: stripe.Int64(request.Plan.PriceCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(request.Plan.Name),
						Description: stripe.String(request.Plan.Description),
					},
				},
			},
		},
		Metadata:         request.Metadata,
		CustomerCreation: stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways)),
	}
	if request.UserEmail != "" {
		params.CustomerEmail = stripe.String(request.UserEmail)
	}

	session, err := stripeSession.New(params)
	if err != nil {
		return nil, err
	}

	return &billingGatewaySession{
		ID:            session.ID,
		URL:           session.URL,
		Status:        string(session.Status),
		PaymentStatus: string(session.PaymentStatus),
		AmountTotal:   session.AmountTotal,
		Currency:      string(session.Currency),
		Metadata:      copyMetadata(session.Metadata),
	}, nil
}

func (g stripeBillingGateway) GetCheckoutStatus(_ context.Context, sessionID string) (*billingGatewaySession, error) {
	stripe.Key = g.apiKey

	session, err := stripeSession.Get(sessionID, nil)
	if err != nil {
		return nil, err
	}

	return &billingGatewaySession{
		ID:            session.ID,
		URL:           session.URL,
		Status:        string(session.Status),
		PaymentStatus: string(session.PaymentStatus),
		AmountTotal:   session.AmountTotal,
		Currency:      string(session.Currency),
		Metadata:      copyMetadata(session.Metadata),
	}, nil
}

func (g stripeBillingGateway) ParseWebhook(payload []byte, signature string) (*billingWebhookEvent, error) {
	if g.webhookSecret == "" {
		return nil, fmt.Errorf("%w: STRIPE_WEBHOOK_SECRET is not configured", ErrBillingWebhookUnavailable)
	}

	event, err := stripeWebhook.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, err
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, err
	}

	return &billingWebhookEvent{
		EventID:       event.ID,
		EventType:     string(event.Type),
		SessionID:     session.ID,
		Status:        string(session.Status),
		PaymentStatus: string(session.PaymentStatus),
		AmountTotal:   session.AmountTotal,
		Currency:      string(session.Currency),
		Metadata:      copyMetadata(session.Metadata),
		Raw:           string(payload),
	}, nil
}

func copyMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func cloneGatewaySession(session *billingGatewaySession) *billingGatewaySession {
	if session == nil {
		return nil
	}

	return &billingGatewaySession{
		ID:            session.ID,
		URL:           session.URL,
		Status:        session.Status,
		PaymentStatus: session.PaymentStatus,
		AmountTotal:   session.AmountTotal,
		Currency:      session.Currency,
		Metadata:      copyMetadata(session.Metadata),
	}
}
