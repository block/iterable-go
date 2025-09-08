package main

import (
	"fmt"
	"log"

	iterable_go "github.com/block/iterable-go"
)

func api_catalog_get_all(apiKey string) {
	client := iterable_go.NewClient(apiKey)

	catalogs, err := client.Catalog().All()
	if err != nil {
		log.Fatalf("Error getting catalogs: %v", err)
	}

	fmt.Printf("Found %d catalogs:\n", len(catalogs.CatalogNames))
	fmt.Printf("Total catalogs: %d\n", catalogs.TotalCatalogsCount)
	for _, catalog := range catalogs.CatalogNames {
		fmt.Printf("Catalog: %s\n", catalog)
	}
}
