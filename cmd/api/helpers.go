package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	structValidator "github.com/go-playground/validator/v10"
	"github.com/muyiwadosunmu/hospital-management/internal/validator"
)

var Validate *structValidator.Validate

func init() {
	Validate = structValidator.New(structValidator.WithRequiredStructEnabled())
}

type envelope map[string]interface{}

// The background() helper accepts an arbitrary function as a parameter.
func (app *application) background(fn func()) {
	app.wg.Add(1)
	// Launch a background goroutine.
	go func() {
		// Recover any panic.
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()
		// Execute the arbitrary function that we passed as the parameter.
		fn()
	}()
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Encode the data to JSON, returning the error if there was one.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Limit the size of the request body to 1MB.
	const maxBytes = 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialize the JSON decoder with DisallowUnknownFields option.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the JSON request into the destination.
	if err := dec.Decode(dst); err != nil {
		// Handle specific JSON decoding errors.
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err) // programmer error
		default:
			return err // default case for any other errors
		}
	}

	// Ensure there are no additional JSON values in the request body.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string. If no key exists this
	// will return the empty string "".
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Otherwise return the string.
	return s
}
func (app *application) readDateParam(qs url.Values, key string) (*time.Time, error) {
	raw := qs.Get(key)
	if raw == "" {
		return nil, nil
	}
	// choose the layout your clients will send; commonly just date
	const layout = "2006-01-02"
	t, err := time.Parse(layout, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: must be YYYY-MM-DD", key)
	}
	return &t, nil
}

// The readCSV() helper reads a string value from the query string and then splits it
// into a slice on the comma character. If no matching key could be found, it returns
// the provided default value.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	// Extract the value from the query string.
	csv := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if csv == "" {
		return defaultValue
	}
	// Otherwise parse the value into a []string slice and return it.
	return strings.Split(csv, ",")
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	// Otherwise, return the converted integer value.
	return i
}
