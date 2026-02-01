package cartstore

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
	"google.golang.org/api/iterator"
)

type SpannerCartStore struct {
	client    *spanner.Client
	tableName string
}

func NewSpannerCartStore(ctx context.Context, dbString, tableName string) (*SpannerCartStore, error) {
	client, err := spanner.NewClient(ctx, dbString)
	if err != nil {
		return nil, fmt.Errorf("failed to create spanner client: %w", err)
	}

	return &SpannerCartStore{
		client:    client,
		tableName: tableName,
	}, nil
}

func (s *SpannerCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Get current quantity
		stmt := spanner.Statement{
			SQL: fmt.Sprintf("SELECT quantity FROM %s WHERE userId = @userId AND productId = @productId", s.tableName),
			Params: map[string]interface{}{
				"userId":    userID,
				"productId": productID,
			},
		}

		var currentQuantity int64 // Spanner uses int64 for integers usually
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			currentQuantity = 0
		} else if err != nil {
			return err
		} else {
			if err := row.Column(0, &currentQuantity); err != nil {
				return err
			}
		}

		newQuantity := currentQuantity + int64(quantity)

		// 2. Upsert
		// Spanner InsertOrUpdate is efficient
		m := spanner.InsertOrUpdate(s.tableName,
			[]string{"userId", "productId", "quantity"},
			[]interface{}{userID, productID, newQuantity})

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		return fmt.Errorf("spanner transaction failed: %w", err)
	}

	return nil
}

func (s *SpannerCartStore) GetCart(ctx context.Context, userID string) (*pb.Cart, error) {
	cart := &pb.Cart{UserId: userID}

	stmt := spanner.Statement{
		SQL: fmt.Sprintf("SELECT productId, quantity FROM %s WHERE userId = @userId", s.tableName),
		Params: map[string]interface{}{
			"userId": userID,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("spanner query failed: %w", err)
		}

		var productID string
		var quantity int64
		if err := row.Columns(&productID, &quantity); err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		cart.Items = append(cart.Items, &pb.CartItem{
			ProductId: productID,
			Quantity:  int32(quantity),
		})
	}

	return cart, nil
}

func (s *SpannerCartStore) EmptyCart(ctx context.Context, userID string) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: fmt.Sprintf("DELETE FROM %s WHERE userId = @userId", s.tableName),
			Params: map[string]interface{}{
				"userId": userID,
			},
		}
		numRows, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		_ = numRows // could log this
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to empty cart in spanner: %w", err)
	}
	return nil
}

func (s *SpannerCartStore) Close() {
	s.client.Close()
}
