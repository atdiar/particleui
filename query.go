// Package ui is a library of functions for simple, generic gui development.
package ui

// This file contains the functions and types used to specify a validation scheme for query parameters.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"time"
)

// TODO should default values be available as a fallback if a parameter exists but its value
// is not supported?

// The updated struct to match the extended schema.
type ParamSchema struct {
	Required  bool     `json:"required"`
	Type      string   `json:"type"`
	Pattern   string   `json:"pattern"`
	MinInt    *int     `json:"min_int,omitempty"`
	MaxInt    *int     `json:"max_int,omitempty"`
	MinFloat  *float64 `json:"min_float,omitempty"`
	MaxFloat  *float64 `json:"max_float,omitempty"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	MinDate   string   `json:"min_date,omitempty"` // e.g., "2024-01-01"
	MaxDate   string   `json:"max_date,omitempty"` // e.g., "2024-12-31"
	Options   []string `json:"options,omitempty"`  // this is a list of allowed values for the parameter
}

type ValidationSchema map[string]ParamSchema

func NewValidationSchema() *ValidationSchema {
	m := make(ValidationSchema)
	return &m
}

func (s ValidationSchema) Add(parameter string, schema ParamSchema) ValidationSchema {
	s[parameter] = schema
	return s
}

// AddStringParam adds a string parameter to the validation schema.
func (s ValidationSchema) AddString(parameter string, required bool, pattern string, minLength, maxLength *int, options []string) ValidationSchema {
	s[parameter] = ParamSchema{
		Required:  required,
		Type:      "string",
		Pattern:   pattern,
		MinLength: minLength,
		MaxLength: maxLength,
		Options:   options,
	}
	return s
}

// AddIntegerParam adds an integer parameter to the validation schema.
func (s ValidationSchema) AddInteger(parameter string, required bool, min, max *int) ValidationSchema {
	s[parameter] = ParamSchema{
		Required: required,
		Type:     "integer",
		MinInt:   min,
		MaxInt:   max,
	}
	return s
}

// AddBooleanParam adds a boolean parameter to the validation schema.
func (s ValidationSchema) AddBoolean(parameter string, required bool) ValidationSchema {
	s[parameter] = ParamSchema{
		Required: required,
		Type:     "boolean",
	}
	return s
}

// AddFloatParam adds a float parameter to the validation schema.
func (s ValidationSchema) AddFloat(parameter string, required bool, min, max *float64) ValidationSchema {
	s[parameter] = ParamSchema{
		Required: required,
		Type:     "float",
		MinFloat: min,
		MaxFloat: max,
	}
	return s
}

// AddDateParam adds a date parameter to the validation schema.
func (s ValidationSchema) AddDate(parameter string, required bool, min, max time.Time) ValidationSchema {
	s[parameter] = ParamSchema{
		Required: required,
		Type:     "date",
		MinDate:  min.Format("2006-01-02"),
		MaxDate:  max.Format("2006-01-02"),
	}
	return s
}

// SetOption adds a parameter with a set of allowed options to the validation schema.
func (s ValidationSchema) SetOptions(parameter string, required bool, allowedoptions ...string) ValidationSchema {
	s[parameter] = ParamSchema{
		Required: required,
		Type:     "string",
		Options:  allowedoptions,
	}
	return s
}

func (s ValidationSchema) MarshalJSON() ([]byte, error) {
	// Custom JSON marshaling can be implemented here if needed.
	// For now, we can use the default marshaling.
	return json.Marshal(map[string]ParamSchema(s))
}

func (s *ValidationSchema) UnmarshalJSON(data []byte) error {
	// Custom JSON unmarshaling can be implemented here if needed.
	// For now, we can use the default unmarshaling.
	var temp map[string]ParamSchema
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	*s = ValidationSchema(temp)
	return nil
}

// ValidateQueryParams validates the query parameters against the provided schema.
func ValidateQueryParams(schema ValidationSchema, params url.Values) error {
	// 1. Check for required parameters.
	for paramName, paramSchema := range schema {
		_, exists := params[paramName]
		if paramSchema.Required && !exists {
			return fmt.Errorf("missing required parameter: %s", paramName)
		}
	}

	// 2. Validate parameters that are present.
	for paramName, paramValues := range params {
		paramSchema, exists := schema[paramName]
		if !exists {
			continue
		}
		for _, paramValue := range paramValues {
			if paramSchema.Options != nil && !slices.Contains(paramSchema.Options, paramValue) {
				return fmt.Errorf("parameter %s has an invalid value: %s", paramName, paramValue)
			}
			// 3. Validate based on type and new constraints.
			switch paramSchema.Type {
			case "string":
				// Check for regular expression pattern.
				if paramSchema.Pattern != "" {
					matched, err := regexp.MatchString(paramSchema.Pattern, paramValue)
					if err != nil {
						return fmt.Errorf("invalid regex pattern for %s: %w", paramName, err)
					}
					if !matched {
						return fmt.Errorf("parameter %s with value '%s' does not match pattern", paramName, paramValue)
					}
				}

				// Check for string length.
				if paramSchema.MinLength != nil && len(paramValue) < *paramSchema.MinLength {
					return fmt.Errorf("parameter %s must be at least %d characters long", paramName, *paramSchema.MinLength)
				}
				if paramSchema.MaxLength != nil && len(paramValue) > *paramSchema.MaxLength {
					return fmt.Errorf("parameter %s must be at most %d characters long", paramName, *paramSchema.MaxLength)
				}

				// Check for options.
				if len(paramSchema.Options) > 0 {
					found := false
					for _, option := range paramSchema.Options {
						if paramValue == option {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("parameter %s with value '%s' is not one of the allowed options", paramName, paramValue)
					}
				}

			case "integer":
				val, err := strconv.Atoi(paramValue)
				if err != nil {
					return fmt.Errorf("parameter %s must be an integer, got '%s'", paramName, paramValue)
				}

				// Check for min and max values.
				if paramSchema.MinInt != nil && val < *paramSchema.MinInt {
					return fmt.Errorf("parameter %s must be at least %d", paramName, *paramSchema.MinInt)
				}
				if paramSchema.MaxInt != nil && val > *paramSchema.MaxInt {
					return fmt.Errorf("parameter %s must be at most %d", paramName, *paramSchema.MaxInt)
				}
			case "boolean":
				// Check if the value is a valid boolean string.
				if paramValue != "true" && paramValue != "false" {
					return fmt.Errorf("parameter %s must be 'true' or 'false', got '%s'", paramName, paramValue)
				}
			case "float":
				val, err := strconv.ParseFloat(paramValue, 64)
				if err != nil {
					return fmt.Errorf("parameter %s must be a float, got '%s'", paramName, paramValue)
				}
				// Check for min and max values.
				if paramSchema.MinFloat != nil && val < *paramSchema.MinFloat {
					return fmt.Errorf("parameter %s must be at least %f", paramName, *paramSchema.MinFloat)
				}
				if paramSchema.MaxFloat != nil && val > *paramSchema.MaxFloat {
					return fmt.Errorf("parameter %s must be at most %f", paramName, *paramSchema.MaxFloat)
				}
			case "date":
				t, err := time.Parse("2006-01-02", paramValue)
				if err != nil {
					return fmt.Errorf("parameter %s must be a valid date in YYYY-MM-DD format, got '%s'", paramName, paramValue)
				}
				// Check for min and max dates.
				if paramSchema.MinDate != "" {
					minDate, err := time.Parse("2006-01-02", paramSchema.MinDate)
					if err != nil {
						return fmt.Errorf("invalid min date for parameter %s: %w", paramName, err)
					}
					if t.Before(minDate) {
						return fmt.Errorf("parameter %s must be on or after %s", paramName, paramSchema.MinDate)
					}
				}
				if paramSchema.MaxDate != "" {
					maxDate, err := time.Parse("2006-01-02", paramSchema.MaxDate)
					if err != nil {
						return fmt.Errorf("invalid max date for parameter %s: %w", paramName, err)
					}
					if t.After(maxDate) {
						return fmt.Errorf("parameter %s must be on or before %s", paramName, paramSchema.MaxDate)
					}
				}
			default:
				return fmt.Errorf("unsupported type '%s' for parameter %s", paramSchema.Type, paramName)
			}
		}

	}

	return nil
}
