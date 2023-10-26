package router

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	json "github.com/json-iterator/go"
	"go.uber.org/zap"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/errorapi"
	logger "github.com/dock-tech/munin-exchange-rate-api/pkg/logger"
)

type (
	HTTPResponseWriter interface {
		Write([]byte) (int, error)
	}

	ResponseAdapterError struct {
		Payload *errorapi.ApiError
		TraceId string
		Status  int
	}

	ResponseErrorAdapter struct {
		Data       any
		Error      error
		StatusCode int
	}
)

const (
	RET403   = 403
	RET500   = 500
	RET502   = 502
	RET503   = 503
	RET504   = 504
	RET400   = 400
	RET404   = 404
	RET422   = 422
	RET429   = 429
	TRACE_ID = "Trace-Id"
)

func (r *ResponseAdapterError) Error() string {
	return statusCodeString(r.Status)
}

func statusCodeString(code int) string {
	return fmt.Sprintf(`%v`, code)
}

func ResponseAdapter(ctx context.Context, respWriter HTTPResponseWriter, res ResponseErrorAdapter) error {

	if res.Error != nil {
		status, payload := processHTTPError(ctx, respWriter.(*fiber.Ctx), res.Error)
		err := &ResponseAdapterError{Status: status, Payload: payload}

		switch ctx := respWriter.(type) {
		case *fiber.Ctx:
			err.TraceId = ctx.Get(TRACE_ID)
			errRet := payload.SetId(ctx.Get(TRACE_ID))
			if errRet != nil {
				return &errRet.Error
			}
			return ctx.Status(status).JSON(payload)
		default:
			marshalPayload, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			_, errRet := ctx.Write(marshalPayload)
			if errRet != nil {
				return errRet
			}
		}

		return err
	}

	// ----- Success -----
	switch ctx := respWriter.(type) {
	case *fiber.Ctx:
		ctx.Status(res.StatusCode)
		if res.StatusCode != http.StatusNoContent || res.Data != nil {
			if data, ok := res.Data.([]byte); ok {
				if len(data) == 0 {
					return nil
				}

				ctx.Response().Header.Add("Content-Type", "application/json")
				_, errRet := ctx.Write(res.Data.([]byte))
				if errRet != nil {
					return errRet
				}
				return nil
			}
			ctx.Response().Header.Add("Content-Type", "application/json")
			return ctx.JSON(res.Data)
		}
	default:
		if res.StatusCode != http.StatusNoContent || res.Data != nil {
			if data, ok := res.Data.([]byte); ok && len(data) == 0 {
				return nil
			}
			marshalResData, err := json.Marshal(res.Data)
			if err != nil {
				return err
			}
			_, errRet := ctx.Write(marshalResData)
			if errRet != nil {
				return errRet
			}
		}
	}

	return nil
}

func processHTTPError(ctx context.Context, respWriter *fiber.Ctx, err error) (status int, payload *errorapi.ApiError) {

	requestPayload := make(map[string]any)
	errU := json.Unmarshal(respWriter.Body(), &requestPayload)
	if errU != nil {
		logger.Error(ctx, err.Error())
	}

	payload = errorapi.NewApiError()

	switch err := err.(type) {
	//nolint:typecheck // Function into package
	case *InvalidEntityError:

		logger.Error(ctx,
			err.Error(),
			zap.Any("error", err.Details),
			zap.Any("entity", err.EntityName),
			zap.Any("type", "InvalidEntityError"),
		)

		if len(err.Details) == 1 {
			if _, ok := err.Details[""]; ok {
				payload.
					SetErrorCode(strconv.Itoa(fiber.StatusUnsupportedMediaType)).
					SetErrorDescription("Unsupported Media Type")
				return status, payload
			}
		}

		status = http.StatusBadRequest
		payload.
			SetErrorCode(strconv.Itoa(status)).
			SetErrorDescription(fiber.ErrBadRequest.Message)
		for attr, details := range err.Details {
			payload.AddErrorDetail(strings.ToLower(attr), details...)
		}
	//nolint:typecheck // Function into package
	case *UseCaseError:

		logger.Error(ctx, err.Error(),
			zap.Any("error", err),
			zap.Any("payload", requestPayload),
			zap.Any("type", "UseCaseError"),
		)

		payload.
			SetErrorCode(strconv.Itoa(fiber.StatusUnprocessableEntity)).
			SetErrorDescription(err.Error())
	//nolint:typecheck // Function into package
	case *RepositoryError:

		logger.Error(ctx, err.Error(),
			zap.Any("error", err.InternalError),
			zap.Any("payload", requestPayload),
			zap.Any("type", "RepositoryError"),
		)

		payload.
			SetErrorCode(strconv.Itoa(fiber.StatusUnprocessableEntity)).
			SetErrorDescription(err.Error())
	//nolint:typecheck // Function into package
	case *ServerError:

		logger.Error(ctx, err.Description,
			zap.Any("error", err),
			zap.Any("metadata", err.Metadata),
			zap.Any("payload", requestPayload),
			zap.Any("type", "ServerError"),
		)

		payload.
			SetErrorCode(strconv.Itoa(fiber.StatusUnprocessableEntity)).
			SetErrorDescription(err.Error())
	//nolint:typecheck // Function into package
	case *ExternalIntegrationError:
		var response map[string]any
		json.Unmarshal(err.Data, &response)

		logger.Error(ctx, err.Error(),
			zap.Any("error", err.InternalError),
			zap.Any("code", err.Code),
			zap.Any("response", response),
			zap.Any("request", err.Metadata),
			zap.Any("type", "ExternalIntegrationError"),
		)

		status = err.Code
		if err.Code == 0 {
			status = RET500
		}

		switch status {
		case RET403, RET500, RET502, RET503, RET504:
			payload.
				SetErrorCode(strconv.Itoa(fiber.StatusInternalServerError)).
				SetErrorDescription(fiber.ErrInternalServerError.Message)
		case RET400, RET404, RET422, RET429:
			payload.
				SetErrorCode(strconv.Itoa(status)).
				SetErrorDescription(err.Error())
			json.Unmarshal(err.Data, &payload)
		}
	default:

		logger.Error(ctx, err.Error(),
			zap.Any("error", err),
			zap.Any("payload", requestPayload),
			zap.Any("type", "UnknownError"),
		)

		payload.
			SetErrorCode(strconv.Itoa(fiber.StatusInternalServerError)).
			SetErrorDescription(fiber.ErrInternalServerError.Message)
	}

	return status, payload
}
