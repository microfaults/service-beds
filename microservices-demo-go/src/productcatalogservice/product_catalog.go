package main

import (
	"context"
	"errors"
	"strings"
	"time"
)

type productCatalog struct {
	catalog    Catalog
	productMap map[string]*Product
}

func (p *productCatalog) ListProducts(ctx context.Context) ([]*Product, error) {
	time.Sleep(extraLatency)
	p.parseCatalog()
	return p.catalog.Products, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, id string) (*Product, error) {
	time.Sleep(extraLatency)
	p.parseCatalog()
	if product, ok := p.productMap[id]; ok {
		return product, nil
	}
	return nil, errors.New("no product with ID " + id)
}

func (p *productCatalog) GetProducts(ctx context.Context, ids []string) ([]*Product, error) {
	time.Sleep(extraLatency)
	p.parseCatalog()

	var result []*Product
	for _, id := range ids {
		if product, ok := p.productMap[id]; ok {
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
	p.parseCatalog()
	var ps []*Product
	for _, product := range p.catalog.Products {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(query)) {
			ps = append(ps, product)
		}
	}

	return ps, nil
}

func (p *productCatalog) parseCatalog() {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		err := loadCatalog(&p.catalog)
		if err != nil {
			return
		}
		reloadCatalog = false

		// Build the product map for O(1) lookups
		p.productMap = make(map[string]*Product, len(p.catalog.Products))
		for _, product := range p.catalog.Products {
			p.productMap[product.ID] = product
		}
	}
}
