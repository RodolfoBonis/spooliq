package services

// AsaasCustomerRequest represents a request to create a customer in Asaas
type AsaasCustomerRequest struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	CpfCnpj              string `json:"cpfCnpj"`
	Phone                string `json:"phone,omitempty"`
	MobilePhone          string `json:"mobilePhone,omitempty"`
	Address              string `json:"address,omitempty"`
	AddressNumber        string `json:"addressNumber,omitempty"`
	Complement           string `json:"complement,omitempty"`
	Province             string `json:"province,omitempty"`
	PostalCode           string `json:"postalCode,omitempty"`
	ExternalReference    string `json:"externalReference,omitempty"`
	NotificationDisabled bool   `json:"notificationDisabled,omitempty"`
}

// AsaasCustomerResponse represents the response from Asaas when creating a customer
type AsaasCustomerResponse struct {
	Object               string `json:"object"`
	ID                   string `json:"id"`
	DateCreated          string `json:"dateCreated"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	CpfCnpj              string `json:"cpfCnpj"`
	Phone                string `json:"phone"`
	MobilePhone          string `json:"mobilePhone"`
	Address              string `json:"address"`
	AddressNumber        string `json:"addressNumber"`
	Complement           string `json:"complement"`
	Province             string `json:"province"`
	PostalCode           string `json:"postalCode"`
	ExternalReference    string `json:"externalReference"`
	NotificationDisabled bool   `json:"notificationDisabled"`
}

// AsaasSubscriptionRequest represents a request to create a subscription in Asaas
type AsaasSubscriptionRequest struct {
	Customer          string  `json:"customer"`    // Customer ID
	BillingType       string  `json:"billingType"` // CREDIT_CARD, BOLETO, PIX
	Value             float64 `json:"value"`
	NextDueDate       string  `json:"nextDueDate"` // YYYY-MM-DD
	Cycle             string  `json:"cycle"`       // MONTHLY, WEEKLY, YEARLY
	Description       string  `json:"description,omitempty"`
	EndDate           string  `json:"endDate,omitempty"` // YYYY-MM-DD
	MaxPayments       int     `json:"maxPayments,omitempty"`
	ExternalReference string  `json:"externalReference,omitempty"`
}

// AsaasSubscriptionResponse represents the response from Asaas when creating a subscription
type AsaasSubscriptionResponse struct {
	Object            string  `json:"object"`
	ID                string  `json:"id"`
	DateCreated       string  `json:"dateCreated"`
	Customer          string  `json:"customer"`
	BillingType       string  `json:"billingType"`
	Value             float64 `json:"value"`
	NextDueDate       string  `json:"nextDueDate"`
	Cycle             string  `json:"cycle"`
	Description       string  `json:"description"`
	Status            string  `json:"status"` // ACTIVE, EXPIRED, CANCELLED
	ExternalReference string  `json:"externalReference"`
}

// AsaasSubscriptionUpdateRequest represents a request to update a subscription in Asaas
type AsaasSubscriptionUpdateRequest struct {
	BillingType       string  `json:"billingType,omitempty"` // CREDIT_CARD, BOLETO, PIX
	Value             float64 `json:"value,omitempty"`
	NextDueDate       string  `json:"nextDueDate,omitempty"` // YYYY-MM-DD
	Cycle             string  `json:"cycle,omitempty"`       // MONTHLY, WEEKLY, YEARLY
	Description       string  `json:"description,omitempty"`
	EndDate           string  `json:"endDate,omitempty"` // YYYY-MM-DD
	ExternalReference string  `json:"externalReference,omitempty"`
	UpdatePendingPayments bool `json:"updatePendingPayments,omitempty"` // Update pending payments with new value
}

// AsaasPaymentResponse represents a payment from Asaas
type AsaasPaymentResponse struct {
	Object                string  `json:"object"`
	ID                    string  `json:"id"`
	DateCreated           string  `json:"dateCreated"`
	Customer              string  `json:"customer"`
	Subscription          string  `json:"subscription"`
	Value                 float64 `json:"value"`
	NetValue              float64 `json:"netValue"`
	BillingType           string  `json:"billingType"`
	Status                string  `json:"status"` // PENDING, RECEIVED, CONFIRMED, OVERDUE
	DueDate               string  `json:"dueDate"`
	PaymentDate           *string `json:"paymentDate"`
	InvoiceURL            string  `json:"invoiceUrl"`
	BankSlipURL           string  `json:"bankSlipUrl"`
	TransactionReceiptURL string  `json:"transactionReceiptUrl"`
	ExternalReference     string  `json:"externalReference"`
	Description           string  `json:"description"`
}

// AsaasPaymentListResponse represents a list of payments from Asaas
type AsaasPaymentListResponse struct {
	Object     string                 `json:"object"`
	HasMore    bool                   `json:"hasMore"`
	TotalCount int                    `json:"totalCount"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	Data       []AsaasPaymentResponse `json:"data"`
}

// AsaasErrorResponse represents an error response from Asaas
type AsaasErrorResponse struct {
	Errors []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"errors"`
}

// AsaasWebhookEvent represents a webhook event from Asaas
type AsaasWebhookEvent struct {
	Event   string               `json:"event"` // PAYMENT_CREATED, PAYMENT_RECEIVED, etc.
	Payment AsaasPaymentResponse `json:"payment"`
}
