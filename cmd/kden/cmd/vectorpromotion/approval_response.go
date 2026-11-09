package vectorpromotion

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output"
)

type approvalActionResponse interface {
	StatusCode() int
	GetBody() []byte
	GetJSON403() *apiclient.Forbidden
	GetJSON404() *apiclient.NotFound
	GetJSON409() *apiclient.Conflict
}

func handleApprovalActionResponse(response approvalActionResponse, successMessage, action string) error {
	switch response.StatusCode() {
	case http.StatusNoContent:
		output.PrintMessage(successMessage)
		return nil
	case http.StatusForbidden:
		return errors.New(response.GetJSON403().Error.Message)
	case http.StatusNotFound:
		return errors.New(response.GetJSON404().Error.Message)
	case http.StatusConflict:
		return errors.New(response.GetJSON409().Error.Message)
	default:
		return fmt.Errorf("%s returned HTTP %d: %s", action, response.StatusCode(), string(response.GetBody()))
	}
}
