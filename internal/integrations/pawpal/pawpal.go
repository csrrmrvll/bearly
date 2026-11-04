package pawpal

import (
	"crypto/subtle"
	"fmt"
)

type WebhookOutcome string

const (
	WebhookUnauthorized WebhookOutcome = "unauthorized"
	WebhookMalformed    WebhookOutcome = "malformed"
	WebhookApproved     WebhookOutcome = "approved"
)

type WebhookVerification struct {
	Outcome WebhookOutcome
	OrderID int64
}

func CreateCheckoutURL(orderID int64) string {
	return fmt.Sprintf("https://pawpal.example/checkout?orderId=%d", orderID)
}

func VerifyWebhook(providedKey, expectedKey string, payload any) WebhookVerification {
	if subtle.ConstantTimeCompare([]byte(providedKey), []byte(expectedKey)) != 1 {
		return WebhookVerification{Outcome: WebhookUnauthorized, OrderID: 0}
	}
	payloadRecord, _ := payload.(map[string]any)
	orderID, _ := payloadRecord["orderId"]
	if orderID == nil {
		return WebhookVerification{Outcome: WebhookMalformed, OrderID: 0}
	}
	switch orderID := orderID.(type) {
	case float64:
		if orderID <= 0 || orderID != float64(int64(orderID)) {
			return WebhookVerification{Outcome: WebhookMalformed, OrderID: 0}
		}
	default:
		return WebhookVerification{Outcome: WebhookMalformed, OrderID: 0}
	}
	if payloadRecord["status"] != "approved" {
		return WebhookVerification{Outcome: WebhookMalformed, OrderID: 0}
	}
	return WebhookVerification{Outcome: WebhookApproved, OrderID: int64(orderID.(float64))}
}
