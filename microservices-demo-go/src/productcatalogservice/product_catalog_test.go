package main

import (
	"context"
	"testing"
)

var (
	mockProductCatalog *productCatalog
)

func setupMock() {
	mockProductCatalog = &productCatalog{
		catalog: Catalog{
			Products: []*Product{},
		},
	}

	mockProductCatalog.catalog.Products = append(mockProductCatalog.catalog.Products, &Product{
		ID:   "abc001",
		Name: "Product Alpha One",
	})
	mockProductCatalog.catalog.Products = append(mockProductCatalog.catalog.Products, &Product{
		ID:   "abc002",
		Name: "Product Delta",
	})
	mockProductCatalog.catalog.Products = append(mockProductCatalog.catalog.Products, &Product{
		ID:   "abc003",
		Name: "Product Alpha Two",
	})
	mockProductCatalog.catalog.Products = append(mockProductCatalog.catalog.Products, &Product{
		ID:   "abc004",
		Name: "Product Gamma",
	})

	// Build product map for O(1) lookups
	mockProductCatalog.productMap = make(map[string]*Product, len(mockProductCatalog.catalog.Products))
	for _, p := range mockProductCatalog.catalog.Products {
		mockProductCatalog.productMap[p.ID] = p
	}
}

func TestGetProductExists(t *testing.T) {
	setupMock()
	product, err := mockProductCatalog.GetProduct(context.Background(), "abc003")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := product.Name, "Product Alpha Two"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestGetProductNotFound(t *testing.T) {
	setupMock()
	_, err := mockProductCatalog.GetProduct(context.Background(), "abc005")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got, want := err.Error(), "no product with ID abc005"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestListProducts(t *testing.T) {
	setupMock()
	products, err := mockProductCatalog.ListProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(products), 4; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestSearchProducts(t *testing.T) {
	setupMock()
	products, err := mockProductCatalog.SearchProducts(context.Background(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(products), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestGetProducts(t *testing.T) {
	setupMock()
	products, err := mockProductCatalog.GetProducts(context.Background(), []string{"abc001", "abc003"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(products), 2; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestGetProductsNotFound(t *testing.T) {
	setupMock()
	_, err := mockProductCatalog.GetProducts(context.Background(), []string{"xyz999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
