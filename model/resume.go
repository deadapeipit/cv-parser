package model

type Resume struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Phone      string            `json:"phone"`
	Skills     []string          `json:"skills"`
	LinkedIn   string            `json:"linkedin,omitempty"`
	GitHub     string            `json:"github,omitempty"`
	OtherLinks map[string]string `json:"other_links,omitempty"`
	Education  []Education       `json:"education"`
	Experience []Experience      `json:"experience"`
}
