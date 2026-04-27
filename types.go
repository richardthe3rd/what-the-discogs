package main

type MasterResult struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Year  int    `json:"year"`
	URL   string `json:"url"`
}

type Version struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Label       string   `json:"label"`
	Country     string   `json:"country"`
	Year        string   `json:"released"`
	CatNo       string   `json:"catno"`
	Format      string   `json:"format"`
	FormatDescs []string `json:"format_descriptions"`
	Thumb       string   `json:"thumb"`
	ResourceURL string   `json:"resource_url"`
}

type ReleaseDetail struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	Year        int          `json:"year"`
	Country     string       `json:"country"`
	Labels      []Label      `json:"labels"`
	Formats     []Format     `json:"formats"`
	Identifiers []Identifier `json:"identifiers"`
	Companies   []Company    `json:"companies"`
	Images      []Image      `json:"images,omitempty"`
	Notes       string       `json:"notes"`
	URL         string       `json:"uri"`
}

type Image struct {
	URI    string `json:"uri"`
	Type   string `json:"type"` // "primary" or "secondary"
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Label struct {
	Name     string `json:"name"`
	CatNo    string `json:"catno"`
	EntityID int    `json:"id"`
}

type Format struct {
	Name         string   `json:"name"`
	Qty          string   `json:"qty"`
	Descriptions []string `json:"descriptions"`
	Text         string   `json:"text"`
}

type Identifier struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type Company struct {
	Name           string `json:"name"`
	EntityTypeName string `json:"entity_type_name"`
}

type CollectionInstance struct {
	InstanceID int    `json:"instance_id"`
	FolderID   int    `json:"folder_id"`
	ReleaseID  int    `json:"id"`
	ResourceURL string `json:"resource_url"`
}

type Folder struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
	ResourceURL string `json:"resource_url"`
}

type Identity struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	ResourceURL  string `json:"resource_url"`
	ConsumerName string `json:"consumer_name"`
}

type Pagination struct {
	Page    int `json:"page"`
	Pages   int `json:"pages"`
	PerPage int `json:"per_page"`
	Items   int `json:"items"`
}

type CollectionField struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "textarea", "dropdown", etc.
	Position int    `json:"position"`
	Public   bool   `json:"public"`
}
