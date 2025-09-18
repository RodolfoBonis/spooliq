package entities

import "net/http"

// AppErrorType representa os tipos de erro da aplicação.
type AppErrorType int

// ErrDatabase represents a database error.
const (
	ErrDatabase AppErrorType = iota + 1001
	ErrRepository
	ErrUsecase
	ErrEntity
	ErrModel
	ErrService
	ErrMiddleware
	ErrRoot
	ErrEnvironment
	ErrNotFound
	ErrInvalidToken
	ErrInvalidCredentials
	ErrUnauthorized
)

// AppErrorTypeToString maps AppErrorType to string representations.
var AppErrorTypeToString = map[AppErrorType]string{
	ErrDatabase:           "Database error",
	ErrRepository:         "Repository error",
	ErrUsecase:            "Usecase error",
	ErrEntity:             "Entity error",
	ErrModel:              "Model error",
	ErrService:            "Service error",
	ErrMiddleware:         "Middleware error",
	ErrRoot:               "Root error",
	ErrEnvironment:        "Environment error",
	ErrNotFound:           "Resource not found",
	ErrInvalidToken:       "Invalid token",
	ErrInvalidCredentials: "Invalid credentials",
	ErrUnauthorized:       "Unauthorized",
}

// AppErrorTypeToHTTP maps AppErrorType to HTTP status codes.
var AppErrorTypeToHTTP = map[AppErrorType]int{
	ErrDatabase:           http.StatusInternalServerError,
	ErrRepository:         http.StatusInternalServerError,
	ErrUsecase:            http.StatusInternalServerError,
	ErrEntity:             http.StatusBadRequest,
	ErrModel:              http.StatusBadRequest,
	ErrService:            http.StatusInternalServerError,
	ErrMiddleware:         http.StatusInternalServerError,
	ErrRoot:               http.StatusInternalServerError,
	ErrEnvironment:        http.StatusInternalServerError,
	ErrNotFound:           http.StatusNotFound,
	ErrInvalidToken:       http.StatusUnauthorized,
	ErrInvalidCredentials: http.StatusUnauthorized,
	ErrUnauthorized:       http.StatusUnauthorized,
}
