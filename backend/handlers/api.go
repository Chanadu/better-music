package handlers

import (
	"context"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type jsonOutput[T any] struct {
	Body T
}

type ApiErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func (h *Handler) RegisterAPI(mux *http.ServeMux) huma.API {
	config := huma.DefaultConfig("Better Music API", "1.0.0")

	config.OpenAPIPath = ""
	config.DocsPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil

	api := humago.New(mux, config)

	h.registerAccount(api)
	h.registerSpotify(api)
	h.registerSession(api)
	h.registerSync(api)

	return api
}

func authenticatedResponses(api huma.API) map[string]*huma.Response {
	schemas := api.OpenAPI().Components.Schemas
	errorSchema := schemas.Schema(reflect.TypeFor[ApiErrorResponse](), true, "")

	unauthorizedSchema := &huma.Schema{AnyOf: []*huma.Schema{
		errorSchema,
		schemas.Schema(reflect.TypeFor[MessageResponse](), true, ""),
	}}

	responses := make(map[string]*huma.Response)

	for _, status := range []int{401, 403, 409, 503} {
		schema := errorSchema
		if status == http.StatusUnauthorized {
			schema = unauthorizedSchema
		}

		responses[strconv.Itoa(status)] = &huma.Response{
			Description: http.StatusText(status),
			Content:     map[string]*huma.MediaType{"application/json": {Schema: schema}},
		}
	}

	addProblemResponses(api, responses, http.StatusUnauthorized, http.StatusServiceUnavailable)

	return responses
}

func addProblemResponses(api huma.API, responses map[string]*huma.Response, statuses ...int) {
	schema := api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[huma.ErrorModel](), true, "")

	for _, status := range statuses {
		code := strconv.Itoa(status)
		if responses[code] == nil {
			responses[code] = &huma.Response{
				Description: http.StatusText(status),
				Content:     map[string]*huma.MediaType{},
			}
		}

		responses[code].Content["application/problem+json"] = &huma.MediaType{Schema: schema}
	}
}

func registerRawJSON[I, O any](api huma.API, operation huma.Operation, schema *huma.Schema, handler func(context.Context, *I) (*O, error)) {
	operation.SkipValidateBody = true
	operation.RequestBody = &huma.RequestBody{
		Required: true,
		Content:  map[string]*huma.MediaType{"application/json": {Schema: schema}},
	}

	huma.Register(api, operation, handler)
	delete(operation.RequestBody.Content, "application/octet-stream")
}
