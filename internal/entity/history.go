package entity

import "time"

type HistoryItem struct {
	ID           int       `json:"id"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	RequestBody  string    `json:"request_body"`
	ResponseBody string    `json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
}
