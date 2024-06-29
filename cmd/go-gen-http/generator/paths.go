package generator

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

type Path struct {
	CanonicalName string
	URL           string
	Method        string

	Request  Request
	Response Response
}

type Request struct {
	Name        string
	Headers     *Parameters
	QueryParams *Parameters
	Body        *RequestBody
}

type Response struct {
	Name  string
	Codes []ResponseCode
}

type Parameters struct {
	Values []Parameter
}

type Parameter struct {
	Name     string
	Key      string
	Required bool
}

func CollectPaths(ctx context.Context, paths *v3high.Paths) ([]Path, error) {
	if paths == nil {
		return nil, nil
	}

	result := make([]Path, 0, orderedmap.Len(paths.PathItems))

	// TODO(max): maybe we need to handle rest of the HTTP methods.
	for pair := range orderedmap.Iterate(ctx, paths.PathItems) {
		url := pair.Key()
		pathItem := pair.Value()

		if pathItem.Get != nil {
			path, err := NewPath(ctx, url, http.MethodGet, pathItem.Get)
			if err != nil {
				return nil, fmt.Errorf("could not create GET path %q: %w", url, err)
			}

			result = append(result, path)
		}

		if pathItem.Post != nil {
			path, err := NewPath(ctx, url, http.MethodPost, pathItem.Post)
			if err != nil {
				return nil, fmt.Errorf("could not create POST path %q: %w", url, err)
			}

			result = append(result, path)
		}
	}

	return result, nil
}

func NewPath(ctx context.Context, url, method string, op *v3high.Operation) (Path, error) {
	canonicalName := method + canonize(url)
	requestCanonicalName := canonicalName + "Request"
	responseCanonicalName := canonicalName + "Response"

	headers, queryParams := collectParams(op.Parameters)
	requestBody := collectRequestBody(requestCanonicalName, op.RequestBody)

	responseCodes, err := collectResponseCodes(ctx, op.Responses)
	if err != nil {
		return Path{}, fmt.Errorf("could not collect response codes: %w", err)
	}

	return Path{
		CanonicalName: canonicalName,
		URL:           url,
		Method:        method,
		Request: Request{
			Name:        requestCanonicalName,
			Headers:     headers,
			QueryParams: queryParams,
			Body:        requestBody,
		},
		Response: Response{
			Name:  responseCanonicalName,
			Codes: responseCodes,
		},
	}, nil
}

func collectParams(params []*v3high.Parameter) (headers, queryParams *Parameters) {
	hdrs := make([]Parameter, 0, len(params))
	qrprms := make([]Parameter, 0, len(params))

	for _, param := range params {
		parameter := Parameter{
			Name:     canonize(param.Name),
			Key:      param.Name,
			Required: resolveptr(param.Required),
		}

		switch param.In {
		case "header":
			hdrs = append(hdrs, parameter)
		case "query":
			qrprms = append(qrprms, parameter)
		}
	}

	if len(hdrs) > 0 {
		headers = &Parameters{Values: hdrs}
	}

	if len(qrprms) > 0 {
		queryParams = &Parameters{Values: qrprms}
	}

	return
}

// TODO(max): Need to implement different content types. E.g. VSS uses "vnd.api
// + application/json" which is basically the same but will fail here.
func collectRequestBody(
	requestCanonicalName string,
	body *v3high.RequestBody,
) *RequestBody {
	if body == nil {
		return nil
	}

	result := &RequestBody{
		Name:     requestCanonicalName + "RequestBody",
		Required: resolveptr(body.Required),
	}

	media := body.Content.GetOrZero("application/json")
	if media == nil || media.Schema == nil {
		return result
	}

	reference := media.Schema.GetReference()
	splits := strings.Split(reference, "/")

	result.Name = splits[len(splits)-1]

	return result
}

// TODO(max): Need to implement different content types support.
func collectResponseCodes(
	ctx context.Context,
	responses *v3high.Responses,
) ([]ResponseCode, error) {
	if responses == nil {
		return nil, nil
	}

	result := make([]ResponseCode, 0, orderedmap.Len(responses.Codes))

	for code := range orderedmap.Iterate(ctx, responses.Codes) {
		httpcode, err := strconv.ParseInt(code.Key(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse code %q: %w", code.Key(), err)
		}

		if httpcode >= http.StatusBadRequest {
			continue
		}

		schema := code.Value()

		media := schema.Content.GetOrZero("application/json")
		if media == nil || media.Schema == nil {
			return nil, fmt.Errorf("invalid response schema %q: %w", code.Key(), err)
		}

		reference := media.Schema.GetReference()
		splits := strings.Split(reference, "/")

		result = append(result, ResponseCode{
			Code: int(httpcode),
			Name: splits[len(splits)-1],
		})
	}

	return result, nil
}
