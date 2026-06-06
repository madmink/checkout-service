package httpresponse

import (
	"checkout-service/app/model/response"
	"checkout-service/app/util"
	"checkout-service/log"
	"context"
	"net/http"
	"strings"
)

const (
	ServiceErrorDefault = "The system is currently experiencing issues. Please try again later"
)

func GetErrorMessage(err string) string {
	errorMessage := strings.Split(err, " | ")
	return strings.TrimSpace(errorMessage[len(errorMessage)-1])
}

func ErrorHandler(ctx context.Context, err error, status int, internalMessage string, resp *response.GeneralResponse) {
	if internalMessage != "" {
		resp.SetError(internalMessage)
	} else {
		resp.SetError(GetErrorMessage(err.Error()))
	}

	if status >= http.StatusInternalServerError {
		if internalMessage != "" {
			resp.SetError(internalMessage)
		}

		log.Logging.Error.Errorln(util.RequestIDLog(ctx) + err.Error())
		return
	}

	log.Logging.Access.Warnln(util.RequestIDLog(ctx) + err.Error())
}
