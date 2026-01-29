package main

// Ad represents an advertisement to be displayed.
type Ad struct {
	RedirectURL string `json:"redirect_url"`
	Text        string `json:"text"`
}

// AdRequest represents the request body for getting ads.
type AdRequest struct {
	// ContextKeys are keywords from the current page to contextually select ads.
	ContextKeys []string `json:"context_keys"`
}

// AdResponse represents the response body containing ads.
type AdResponse struct {
	Ads []Ad `json:"ads"`
}
