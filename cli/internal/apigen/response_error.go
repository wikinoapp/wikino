package apigen

import "fmt"

// ResponseErrorは、送信済みの要求の応答を読み取り・解析できなかったことを表す。
// 書き込みが反映された可能性があるため、再送前の確認を呼び出し側で案内する。
type ResponseError struct {
	StatusCode int
	Err        error
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("HTTP %dの応答を読み取り・解析できませんでした: %v", e.StatusCode, e.Err)
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}
