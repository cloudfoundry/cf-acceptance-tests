package health

import (
	"fmt"
	"io"
	"net/http"
)

var callCount = 0

func HealthHander(res http.ResponseWriter, req *http.Request) {
	if callCount < 3 {
		callCount++
		res.WriteHeader(http.StatusInternalServerError)
		io.WriteString(res, fmt.Sprintf("Hit /health %d times", callCount))
		return
	}

	io.WriteString(res, "I'm alive")
}
