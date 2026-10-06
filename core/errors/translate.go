package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// translateValidationErrors converts validator.ValidationErrors into a
// field->message map with pt-BR messages. Field keys come from the json tag
// name (the shared validator registers a tag-name func), so they match the
// request/response payloads clients actually send.
func translateValidationErrors(verrs validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fe.Field()] = translateFieldError(fe)
	}
	return fields
}

// translateFieldError returns a pt-BR message for a single field error. The
// wording for size-related tags (min, max, len) depends on the field kind:
// strings talk about "caracteres", collections about "itens", and numbers use
// a plain numeric comparison.
func translateFieldError(fe validator.FieldError) string {
	param := fe.Param()
	switch fe.Tag() {
	case "required":
		return "obrigatório"
	case "email":
		return "deve ser um e-mail válido"
	case "uuid", "uuid4":
		return "deve ser um UUID válido"
	case "url":
		return "deve ser uma URL válida"
	case "oneof":
		return fmt.Sprintf("deve ser um dos valores: %s", param)
	case "min":
		return sizeMessage(fe.Kind(), "no mínimo", param)
	case "max":
		return sizeMessage(fe.Kind(), "no máximo", param)
	case "len":
		return exactLenMessage(fe.Kind(), param)
	case "gt":
		return fmt.Sprintf("deve ser maior que %s", param)
	case "gte":
		return fmt.Sprintf("deve ser maior ou igual a %s", param)
	case "lt":
		return fmt.Sprintf("deve ser menor que %s", param)
	case "lte":
		return fmt.Sprintf("deve ser menor ou igual a %s", param)
	default:
		return "inválido"
	}
}

func sizeMessage(kind reflect.Kind, bound, param string) string {
	switch kind {
	case reflect.String:
		return fmt.Sprintf("deve ter %s %s caracteres", bound, param)
	case reflect.Slice, reflect.Array, reflect.Map:
		return fmt.Sprintf("deve ter %s %s itens", bound, param)
	default:
		return fmt.Sprintf("deve ser %s %s", bound, param)
	}
}

func exactLenMessage(kind reflect.Kind, param string) string {
	switch kind {
	case reflect.String:
		return fmt.Sprintf("deve ter exatamente %s caracteres", param)
	case reflect.Slice, reflect.Array, reflect.Map:
		return fmt.Sprintf("deve ter exatamente %s itens", param)
	default:
		return fmt.Sprintf("deve ser exatamente %s", param)
	}
}

// isBindingError reports whether err looks like a request-decoding failure
// (malformed JSON, type mismatches, unknown fields) rather than a domain error.
// These map to 400 invalid_request.
func isBindingError(err error) bool {
	var (
		jsonSyntax       *json.SyntaxError
		jsonType         *json.UnmarshalTypeError
		invalidUnmarshal *json.InvalidUnmarshalError
	)
	if stderrors.As(err, &jsonSyntax) ||
		stderrors.As(err, &jsonType) ||
		stderrors.As(err, &invalidUnmarshal) {
		return true
	}
	// io.EOF / unexpected EOF surface as plain errors from gin's binding; match
	// them by message to avoid a hard dependency on gin's internal error types.
	switch err.Error() {
	case "EOF", "unexpected EOF", "invalid request":
		return true
	}
	return false
}
