package quotes

import (
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	_ "github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto" // Required for Swagger documentation
	"github.com/gin-gonic/gin"
)

// CreateQuoteHandler handles creating a new quote.
// @Summary Create a new quote
// @Schemes
// @Description Create a new quote with filament lines and profiles
// @Tags Quotes
// @Accept json
// @Produce json
// @Param quote body dto.CreateQuoteRequest true "Quote data"
// @Success 201 {object} dto.QuoteResponse "Successfully created quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes [post]
// @Security Bearer
func CreateQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.CreateQuote(c)
	}
}

// GetQuoteHandler handles getting a quote by ID.
// @Summary Get a quote by ID
// @Schemes
// @Description Get a quote by ID with all its filament lines and profiles
// @Tags Quotes
// @Produce json
// @Param id path int true "Quote ID"
// @Success 200 {object} dto.QuoteResponse "Successfully retrieved quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /quotes/{id} [get]
// @Security Bearer
func GetQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.GetQuote(c)
	}
}

// GetUserQuotesHandler handles getting all quotes for the authenticated user.
// @Summary Get all quotes for the authenticated user
// @Schemes
// @Description Get all quotes belonging to the authenticated user
// @Tags Quotes
// @Produce json
// @Success 200 {object} map[string]interface{} "Successfully retrieved user quotes"
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes [get]
// @Security Bearer
func GetUserQuotesHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.GetUserQuotes(c)
	}
}

// UpdateQuoteHandler handles updating a quote.
// @Summary Update a quote
// @Schemes
// @Description Update a quote with new data
// @Tags Quotes
// @Accept json
// @Produce json
// @Param id path int true "Quote ID"
// @Param quote body dto.UpdateQuoteRequest true "Quote data"
// @Success 200 {object} dto.QuoteResponse "Successfully updated quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes/{id} [put]
// @Security Bearer
func UpdateQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.UpdateQuote(c)
	}
}

// DeleteQuoteHandler handles deleting a quote.
// @Summary Delete a quote
// @Schemes
// @Description Delete a quote and all its related data
// @Tags Quotes
// @Param id path int true "Quote ID"
// @Success 204 "Successfully deleted quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes/{id} [delete]
// @Security Bearer
func DeleteQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.DeleteQuote(c)
	}
}

// DuplicateQuoteHandler handles duplicating a quote.
// @Summary Duplicate a quote
// @Schemes
// @Description Create a copy of an existing quote
// @Tags Quotes
// @Produce json
// @Param id path int true "Quote ID"
// @Success 201 {object} dto.QuoteResponse "Successfully duplicated quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes/{id}/duplicate [post]
// @Security Bearer
func DuplicateQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.DuplicateQuote(c)
	}
}

// CalculateQuoteHandler handles calculating quote costs.
// @Summary Calculate quote costs
// @Schemes
// @Description Calculate all costs for a quote based on provided parameters
// @Tags Quotes
// @Accept json
// @Produce json
// @Param id path int true "Quote ID"
// @Param calculation body dto.CalculateQuoteRequest true "Calculation parameters"
// @Success 200 {object} dto.CalculationResult "Successfully calculated quote"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /quotes/{id}/calculate [post]
// @Security Bearer
func CalculateQuoteHandler(quoteUc usecases.QuoteUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		quoteUc.CalculateQuote(c)
	}
}

// Routes registers quote routes for the application.
func Routes(route *gin.RouterGroup, quoteUC usecases.QuoteUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	quotes := route.Group("/quotes")

	// Protected routes (require authentication)
	quotes.POST("", protectFactory(CreateQuoteHandler(quoteUC), "user"))
	quotes.GET("", protectFactory(GetUserQuotesHandler(quoteUC), "user"))
	quotes.GET("/:id", protectFactory(GetQuoteHandler(quoteUC), "user"))
	quotes.PUT("/:id", protectFactory(UpdateQuoteHandler(quoteUC), "user"))
	quotes.DELETE("/:id", protectFactory(DeleteQuoteHandler(quoteUC), "user"))
	quotes.POST("/:id/duplicate", protectFactory(DuplicateQuoteHandler(quoteUC), "user"))
	quotes.POST("/:id/calculate", protectFactory(CalculateQuoteHandler(quoteUC), "user"))
}
