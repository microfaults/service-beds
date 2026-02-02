package main

import (
	"encoding/json"
	"os"
)

func loadCatalog(catalog *Catalog) error {
	catalogMutex.Lock()
	defer catalogMutex.Unlock()

	return loadCatalogFromLocalFile(catalog)
}

func loadCatalogFromLocalFile(catalog *Catalog) error {
	log.Info("loading catalog from local products.json file...")

	catalogJSON, err := os.ReadFile("products.json")
	if err != nil {
		log.Warnf("failed to open product catalog json file: %v", err)
		return err
	}

	if err := json.Unmarshal(catalogJSON, catalog); err != nil {
		log.Warnf("failed to parse the catalog JSON: %v", err)
		return err
	}

	log.Info("successfully parsed product catalog json")
	return nil
}
