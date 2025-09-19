package handlers

import (
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"github.com/gin-gonic/gin"
)

type QuoteHandler struct {
	quoteUseCase usecases.QuoteUseCase
}

func NewQuoteHandler(quoteUseCase usecases.QuoteUseCase) *QuoteHandler {
	return &QuoteHandler{
		quoteUseCase: quoteUseCase,
	}
}

// CreateQuote godoc
// @Summary Create a new quote
// @Description Create a new quote with filament lines and profiles
// @Tags quotes
// @Accept json
// @Produce json
// @Param quote body usecases.CreateQuoteRequest true "Quote data"
// @Success 201 {object} usecases.QuoteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes [post]
// @Security BearerAuth
func (h *QuoteHandler) CreateQuote(c *gin.Context) {
	h.quoteUseCase.CreateQuote(c)
}

// GetQuote godoc
// @Summary Get a quote by ID
// @Description Get a quote by ID with all its filament lines and profiles
// @Tags quotes
// @Produce json
// @Param id path int true "Quote ID"
// @Success 200 {object} usecases.QuoteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /v1/quotes/{id} [get]
// @Security BearerAuth
func (h *QuoteHandler) GetQuote(c *gin.Context) {
	h.quoteUseCase.GetQuote(c)
}

// GetUserQuotes godoc
// @Summary Get all quotes for the authenticated user
// @Description Get all quotes belonging to the authenticated user
// @Tags quotes
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes [get]
// @Security BearerAuth
func (h *QuoteHandler) GetUserQuotes(c *gin.Context) {
	h.quoteUseCase.GetUserQuotes(c)
}

// UpdateQuote godoc
// @Summary Update a quote
// @Description Update a quote with new data
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path int true "Quote ID"
// @Param quote body usecases.UpdateQuoteRequest true "Quote data"
// @Success 200 {object} usecases.QuoteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes/{id} [put]
// @Security BearerAuth
func (h *QuoteHandler) UpdateQuote(c *gin.Context) {
	h.quoteUseCase.UpdateQuote(c)
}

// DeleteQuote godoc
// @Summary Delete a quote
// @Description Delete a quote and all its related data
// @Tags quotes
// @Param id path int true "Quote ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes/{id} [delete]
// @Security BearerAuth
func (h *QuoteHandler) DeleteQuote(c *gin.Context) {
	h.quoteUseCase.DeleteQuote(c)
}

// DuplicateQuote godoc
// @Summary Duplicate a quote
// @Description Create a copy of an existing quote
// @Tags quotes
// @Produce json
// @Param id path int true "Quote ID"
// @Success 201 {object} usecases.QuoteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes/{id}/duplicate [post]
// @Security BearerAuth
func (h *QuoteHandler) DuplicateQuote(c *gin.Context) {
	h.quoteUseCase.DuplicateQuote(c)
}

// CalculateQuote godoc
// @Summary Calculate quote costs
// @Description Calculate all costs for a quote based on provided parameters
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path int true "Quote ID"
// @Param calculation body usecases.CalculateQuoteRequest true "Calculation parameters"
// @Success 200 {object} usecases.CalculationResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/quotes/{id}/calculate [post]
// @Security BearerAuth
func (h *QuoteHandler) CalculateQuote(c *gin.Context) {
	h.quoteUseCase.CalculateQuote(c)
}