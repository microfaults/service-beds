package main

import (
	"context"
	"errors"
	"strings"
	"time"
)

type productCatalog struct {
	catalog Catalog
}

func (p *productCatalog) ListProducts(ctx context.Context) ([]*Product, error) {
	time.Sleep(extraLatency)
	return p.parseCatalog(), nil
}

func (p *productCatalog) GetProduct(ctx context.Context, id string) (*Product, error) {
	time.Sleep(extraLatency)
	var found *Product
	products := p.parseCatalog()
	for i := 0; i < len(products); i++ {
		if id == products[i].ID {
			found = products[i]
		}
	}

	if found == nil {
		return nil, errors.New("no product with ID " + id)
	}
	return found, nil
}

func (p *productCatalog) GetProducts(ctx context.Context, ids []string) ([]*Product, error) {
	time.Sleep(extraLatency)
	products := p.parseCatalog()

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	var result []*Product
	for _, product := range products {
		if idSet[product.ID] {
			result = append(result, product)
		}
	}

	if len(result) == 0 {
		return nil, errors.New("no products found for the given IDs")
	}
	return result, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, query string) ([]*Product, error) {
	time.Sleep(extraLatency)
	var ps []*Product
	products := p.parseCatalog()
	for _, product := range products {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(query)) {
			ps = append(ps, product)
		}
	}

	return ps, nil
}

func (p *productCatalog) parseCatalog() []*Product {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		err := loadCatalog(&p.catalog)
		if err != nil {
			return []*Product{}
		}
		reloadCatalog = false
	}

	return p.catalog.Products
}
