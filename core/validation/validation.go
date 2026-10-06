// Package validation exposes a single, shared validator configured to report
// field names from the json struct tag. It is both callable directly
// (validation.Validate) and registered as gin's binding validator so that
// `binding:` and `validate:` struct tags are both enforced.
package validation

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// validate is the canonical validator used by Validate(). It reads the
// `validate:` tag (the go-playground default) and reports json field names.
var validate *validator.Validate

// bindingValidate mirrors validate but reads the `binding:` tag, so existing
// gin `binding:"required"` annotations keep working when this package replaces
// gin's default binding validator.
var bindingValidate *validator.Validate

var once sync.Once

func init() {
	validate = newValidator("validate")
	bindingValidate = newValidator("binding")
}

// newValidator builds a validator.Validate that uses the given struct tag and
// reports the json field name (first segment of the json tag) in errors.
func newValidator(tag string) *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.SetTagName(tag)
	v.RegisterTagNameFunc(jsonFieldName)
	return v
}

func jsonFieldName(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return fld.Name
	}
	return name
}

// Validate validates s against its `validate:` tags and returns a
// validator.ValidationErrors on failure (which errors.Respond translates to a
// pt-BR, per-field envelope). It returns nil when s is valid.
func Validate(s any) error {
	return validate.Struct(s)
}

// Instance returns the shared validator (reading the `validate:` tag). Use it
// to register custom validations once at startup.
func Instance() *validator.Validate {
	return validate
}

// Register installs this package as gin's binding validator. It is idempotent
// and should be called once during application startup, before routes handle
// requests. After registration, gin's ShouldBind/Bind enforce both `binding:`
// and `validate:` tags and report json field names.
func Register() {
	once.Do(func() {
		binding.Validator = &ginValidator{}
	})
}

// ginValidator adapts the shared validators to gin's StructValidator contract.
// It enforces `binding:` tags first (preserving legacy behavior) and then
// `validate:` tags, so both are honored on a single ShouldBind call.
type ginValidator struct{}

// ValidateStruct validates obj, unwrapping pointers and slices as gin does.
func (v *ginValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		return v.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		if err := bindingValidate.Struct(obj); err != nil {
			return err
		}
		return validate.Struct(obj)
	case reflect.Slice, reflect.Array:
		var errs []error
		for i := 0; i < value.Len(); i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	default:
		return nil
	}
}

// Engine returns the underlying binding validator so callers that type-assert
// gin's validator engine (to register custom validations) keep working.
func (v *ginValidator) Engine() any {
	return bindingValidate
}
