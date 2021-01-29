// Package errors implements functions to manipulate errors.
package errors

import (
	"encoding/json"
)

// Linux exited code to custom error
var ExitedCode2JudgerError = map[int64]JudgerError{
	0:   NOTHING,
	2:   RE,
	126: ENV,
	137: DELETE, // SIGKILL
	143: TLE,    // SIGTERM terminated by timeout
}

type JudgerError int

const (
	NOTHING JudgerError = iota
	CE
	RE
	TLE
	ENV // environment error
	DELETE
	OutputNotFound
	AnswerNotFound
	UNKNOWN
)

type Err struct {
	Code JudgerError
	Msg  string
}

func (e Err) Error() string {
	err, _ := json.Marshal(e)
	return string(err)
}

func New(code JudgerError, msg string) Err {
	return Err{
		Code: code,
		Msg:  msg,
	}
}

func IsError(err error, judgerError JudgerError) bool {
	e, ok := err.(Err)
	if !ok {
		return false
	}
	return e.Code == judgerError
}
