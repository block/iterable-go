package types

type Catalogs struct {
	CatalogNames       []string `json:"catalogNames"`
	TotalCatalogsCount int      `json:"totalCatalogsCount"`
	PreviousPageUrl    string   `json:"previousPageUrl,omitempty"`
	NextPageUrl        string   `json:"nextPageUrl,omitempty"`
}

type CatalogFieldMapping struct {
	DefinedMappings map[string]interface{} `json:"definedMappings"`
	UndefinedFields []string               `json:"undefinedFields"`
}
