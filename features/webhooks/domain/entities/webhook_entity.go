package entities

// AsaasWebhookEvent represents an incoming webhook event from Asaas
type AsaasWebhookEvent struct {
	Event   string                 `json:"event"`
	Payment map[string]interface{} `json:"payment"`
}

// AsaasWebhookRequest represents the full webhook request from Asaas
type AsaasWebhookRequest struct {
	Event        string                   `json:"event"`
	Payment      AsaasPaymentWebhook      `json:"payment,omitempty"`
	Subscription AsaasSubscriptionWebhook `json:"subscription,omitempty"`
}

// AsaasPaymentWebhook represents payment data from webhook
type AsaasPaymentWebhook struct {
	ID                string  `json:"id"`
	Customer          string  `json:"customer"`
	Subscription      string  `json:"subscription"`
	Value             float64 `json:"value"`
	NetValue          float64 `json:"netValue"`
	Status            string  `json:"status"`
	BillingType       string  `json:"billingType"`
	DueDate           string  `json:"dueDate"`
	PaymentDate       string  `json:"paymentDate"`
	ClientPaymentDate string  `json:"clientPaymentDate"`
	Description       string  `json:"description"`
	ExternalReference string  `json:"externalReference"`
	InvoiceURL        string  `json:"invoiceUrl"`
}

// AsaasSubscriptionWebhook represents subscription data from webhook
type AsaasSubscriptionWebhook struct {
	ID                string  `json:"id"`
	Customer          string  `json:"customer"`
	Value             float64 `json:"value"`
	Status            string  `json:"status"`
	BillingType       string  `json:"billingType"`
	Cycle             string  `json:"cycle"`
	Description       string  `json:"description"`
	ExternalReference string  `json:"externalReference"`
}
