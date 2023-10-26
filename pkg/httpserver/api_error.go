package httpserver

import (
	"fmt"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
)

type DockApiError struct {
	InnerError dockApiInnerError `json:"error"`
	StatusCode int               `json:"-"`
}

type dockApiInnerError struct {
	Id           string               `json:"id"`
	Code         string               `json:"code"`
	Description  string               `json:"description"`
	ErrorDetails []dockApiErrorDetail `json:"error_details,omitempty"`
}

type dockApiErrorDetail struct {
	Attribute string   `json:"attribute"`
	Messages  []string `json:"messages"`
}

func (dae *DockApiError) Error() string {
	if len(dae.InnerError.ErrorDetails) > 0 {
		return fmt.Sprintf("id=%s,code=%s,description=%s,error_details=%s", dae.InnerError.Id, dae.InnerError.Code, dae.InnerError.Description, dae.InnerError.ErrorDetails)
	}
	return fmt.Sprintf("id=%s,code=%s,description=%s", dae.InnerError.Id, dae.InnerError.Code, dae.InnerError.Description)
}

func MakeDockApiErrorCode(baseCode, detailCode string) string {
	return fmt.Sprintf("%s-%s", baseCode, detailCode)
}

func NewDockApiError(statusCode int, code, description string) *DockApiError {
	return &DockApiError{
		StatusCode: statusCode,
		InnerError: dockApiInnerError{
			Id:          uuid.NewString(),
			Code:        code,
			Description: description,
		},
	}
}

func (dae *DockApiError) AddErrorDetail(attribute string, messages ...string) {
	if dae.InnerError.ErrorDetails == nil {
		dae.InnerError.ErrorDetails = make([]dockApiErrorDetail, 0)
	}

	ed := dockApiErrorDetail{
		Attribute: attribute,
		Messages:  messages,
	}

	dae.InnerError.ErrorDetails = append(dae.InnerError.ErrorDetails, ed)
}

func (dae *DockApiError) JsonMap() map[string]any {
	var m map[string]any

	b, _ := json.Marshal(&dae)
	json.Unmarshal(b, &m)

	return m
}
