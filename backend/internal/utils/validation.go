package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Превращаем ошибки биндинга/валидации в короткие понятные сообщения на русском.
func FormatRequestBindError(err error, obj interface{}) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, io.EOF) {
		return "Тело запроса пустое"
	}
	var syn *json.SyntaxError
	if errors.As(err, &syn) {
		return "Некорректный JSON: синтаксическая ошибка"
	}
	var unmarshal *json.UnmarshalTypeError
	if errors.As(err, &unmarshal) {
		name := jsonNameForStructField(obj, unmarshal.Field)
		return fmt.Sprintf("Поле %s имеет неверный формат", name)
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		var parts []string
		for _, fe := range verrs {
			parts = append(parts, formatValidationError(fe, obj))
		}
		return strings.Join(parts, "; ")
	}
	return err.Error()
}

func jsonNameForStructField(obj interface{}, structFieldName string) string {
	t := reflect.TypeOf(obj)
	if t == nil {
		return structFieldName
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	f, ok := t.FieldByName(structFieldName)
	if !ok {
		return strings.ToLower(structFieldName[:1]) + structFieldName[1:]
	}
	tag := strings.Split(f.Tag.Get("json"), ",")[0]
	if tag == "" || tag == "-" {
		return structFieldName
	}
	return tag
}

func formatValidationError(fe validator.FieldError, obj interface{}) string {
	name := jsonNameForStructField(obj, fe.StructField())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Поле %s обязательно", name)
	case "min":
		return fmt.Sprintf("Поле %s имеет неверный формат", name)
	case "max":
		return fmt.Sprintf("Поле %s имеет неверный формат", name)
	case "oneof":
		return fmt.Sprintf("Поле %s имеет недопустимое значение", name)
	default:
		return fmt.Sprintf("Поле %s не прошло проверку (%s)", name, fe.Tag())
	}
}
