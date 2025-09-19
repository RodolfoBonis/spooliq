package usecases

import (
	"github.com/gin-gonic/gin"
)

// QuoteUseCase define as operações de negócio para orçamentos
type QuoteUseCase interface {
	CreateQuote(c *gin.Context)
	GetQuote(c *gin.Context)
	GetUserQuotes(c *gin.Context)
	UpdateQuote(c *gin.Context)
	DeleteQuote(c *gin.Context)
	DuplicateQuote(c *gin.Context)
	CalculateQuote(c *gin.Context)
}