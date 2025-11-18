package model

type URL struct {
	UUID     string `json:"uuid"`
	Short    string `json:"short_url"`
	Original string `json:"original_url"`
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}
