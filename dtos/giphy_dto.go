package dtos

type GiphyAutocompleteItem struct {
	Name string `json:"name"`
}

type GiphyAutocompleteResponse struct {
	Data []GiphyAutocompleteItem `json:"data"`
}

type GiphyTemplate struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

type GiphyTemplatesResponse struct {
	Data []GiphyTemplate `json:"data"`
}
