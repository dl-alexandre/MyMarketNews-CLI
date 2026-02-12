package util

import "encoding/json"

type statsEnvelope struct {
	Stats statsData `json:"stats"`
}

type statsData struct {
	TotalRows       int `json:"totalRows"`
	ReturnedRows    int `json:"returnedRows"`
	UserAllowedRows int `json:"userAllowedRows"`
}

func IsResponseTruncated(body []byte) bool {
	var envelope statsEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	if envelope.Stats.UserAllowedRows == 0 {
		return false
	}
	if envelope.Stats.ReturnedRows >= envelope.Stats.UserAllowedRows {
		return true
	}
	if envelope.Stats.TotalRows > 0 && envelope.Stats.ReturnedRows > 0 && envelope.Stats.TotalRows > envelope.Stats.ReturnedRows {
		return true
	}
	return false
}
