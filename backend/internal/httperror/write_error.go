package httperror

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func WriteError(c *gin.Context, err error) {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)

	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status < 400 || appErr.Status > 599 {
		log.Printf("request_id=%s error=%v", requestID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:      "INTERNAL",
			Message:   "internal error",
			Fields:    map[string]string{},
			RequestID: requestID,
		})
		return
	}
	fields := appErr.Fields
	if fields == nil {
		fields = map[string]string{}
	}
	c.JSON(appErr.Status, ErrorResponse{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Fields:    fields,
		RequestID: requestID,
	})
}
