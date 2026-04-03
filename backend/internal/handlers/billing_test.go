package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/tests"
)

func TestBillingHandler_GetPlans(t *testing.T) {
	db := tests.RequireDB(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	handler := &BillingHandler{
		db:      database.Wrap(db),
		gateway: newMockBillingGateway(),
	}

	app.Get("/api/v1/billing/plans", handler.GetPlans)

	req := httptest.NewRequest("GET", "/api/v1/billing/plans", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Plans []billingPlanResponse `json:"plans"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	require.Len(t, body.Plans, 2)
	assert.Equal(t, "pro", body.Plans[0].ID)
	assert.Equal(t, "team", body.Plans[1].ID)
}

func TestBillingHandler_CreateCheckoutAndGetStatus(t *testing.T) {
	db := tests.RequireDB(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	handler := &BillingHandler{
		db:      database.Wrap(db),
		gateway: newMockBillingGateway(),
	}

	app.Post("/api/v1/billing/checkout", handler.CreateCheckout)
	app.Get("/api/v1/billing/status/:session_id", handler.GetStatus)

	body := []byte(`{"plan_id":"pro","origin_url":"http://localhost:3001","user_email":"launchtester@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/billing/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var checkout billingCheckoutResponse
	err = json.NewDecoder(resp.Body).Decode(&checkout)
	require.NoError(t, err)
	assert.NotEmpty(t, checkout.SessionID)
	assert.Contains(t, checkout.URL, "/pricing?session_id=")

	statusReq := httptest.NewRequest("GET", "/api/v1/billing/status/"+checkout.SessionID, nil)
	statusResp, err := app.Test(statusReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, statusResp.StatusCode)

	var status billingStatusResponse
	err = json.NewDecoder(statusResp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, checkout.SessionID, status.SessionID)
	assert.Equal(t, "complete", status.Status)
	assert.Equal(t, "paid", status.PaymentStatus)
	assert.Equal(t, int64(2900), status.AmountTotal)
	assert.Equal(t, "usd", status.Currency)
	require.NotNil(t, status.Transaction)
	assert.Equal(t, "pro", status.Transaction["plan_id"])
	assert.Equal(t, "Pro", status.Transaction["plan_name"])
}

func TestBillingHandler_HandleStripeWebhook_Unconfigured(t *testing.T) {
	db := tests.RequireDB(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	handler := &BillingHandler{
		db:      database.Wrap(db),
		gateway: disabledBillingGateway{reason: "STRIPE_WEBHOOK_SECRET is not configured"},
	}

	app.Post("/api/webhook/stripe", handler.HandleStripeWebhook)

	req := httptest.NewRequest("POST", "/api/webhook/stripe", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Stripe-Signature", "t=1,v1=signature")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}
