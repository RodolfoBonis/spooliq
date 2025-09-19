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
	ErrConflict
)

// AppErrorTypeToString maps AppErrorType to string representations.
var AppErrorTypeToString = map[AppErrorType]string{
	ErrDatabase:           "Erro de banco de dados",
	ErrRepository:         "Erro de repositório",
	ErrUsecase:            "Erro de caso de uso",
	ErrEntity:             "Erro de entidade",
	ErrModel:              "Erro de modelo",
	ErrService:            "Erro de serviço",
	ErrMiddleware:         "Erro de middleware",
	ErrRoot:               "Erro raiz",
	ErrEnvironment:        "Erro de ambiente",
	ErrNotFound:           "Recurso não encontrado",
	ErrInvalidToken:       "Token inválido",
	ErrInvalidCredentials: "Credenciais inválidas",
	ErrUnauthorized:       "Não autorizado",
	ErrConflict:           "Conflito",
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
	ErrConflict:           http.StatusConflict,
}
