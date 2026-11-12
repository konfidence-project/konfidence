package vectorpromotion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
)

func checkActionResponse(status int, body []byte, action string) error {
	switch status {
	case http.StatusNoContent:
		return nil
	case http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity:
		var response apiclient.ErrorResponse
		if err := json.Unmarshal(body, &response); err == nil &&
			strings.TrimSpace(response.Error.Message) != "" {
			return errors.New(response.Error.Message)
		}
	}

	return fmt.Errorf("%s returned HTTP %d: %s", action, status, body)
}
