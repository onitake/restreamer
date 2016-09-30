import {
	"fmt"
}

const {
	ERR_HTTP_NO_CONNECTION int = -2000
	ERR_HTPP_INVALID_RESPONSE int = -2001
	ERR_QUEUE_FULL int = -1000
	ERR_QUEUE_EMPTY int = -1001
}

type Error struct {
	Code int
	Message string
}

func NewDefError(int code) error {
	return &Error {
		Code: code
	}
}

func NewMsgError(int code, string msg) error {
	return &Error {
		Code: code,
		Message: msg
	}
}

func (e Error) Error() string {
	if (e.Message != nil) {
		return e.Message
	} else {
		switch e.Code {
			case ERR_HTTP_NO_CONNECTION:
				return "Socket not connected"
			case ERR_HTPP_INVALID_RESPONSE:
				return "Unsupported response code"
			case ERR_QUEUE_FULL:
				return "Queue full"
			case ERR_QUEUE_EMPTY:
				return "Queue empty"
			default:
				return fmt.Sprintf("Invalid error code: %d", e.Code)
		}
	}
}
