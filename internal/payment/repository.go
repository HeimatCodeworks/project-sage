package payment

//go:generate mockgen -destination=./repository_mock_test.go -package=payment -source=repository.go Repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-sage/internal/domain"
)

// Repository defines the database operations for the payment service.
type Repository interface {
	// GetProducts fetches all products from the products table
	GetProducts(ctx context.Context) ([]*domain.Product, error)
	// GetProductByID fetches a single product by its ID or Apple/Google ID.
	GetProductByID(ctx context.Context, productID string) (*domain.Product, error)
	// CreateTransaction logs a successful purchase
	CreateTransaction(ctx context.Context, tx *domain.PaymentTransaction) error
}

// postgresRepository is the concrete implementation.
type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository is the constructor for the repository
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// GetProducts fetches all purchasable products from the database.
func (pr *postgresRepository) GetProducts(ctx context.Context) ([]*domain.Product, error) {
	query := `
		SELECT 
			product_id, name, description, price_cents, 
			token_credit, is_subscription, stripe_price_id, 
			apple_product_id, google_product_id
		FROM products
		WHERE is_active = true
		ORDER BY price_cents ASC
	`

	rows, err := pr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("could not query products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ProductID,
			&p.Name,
			&p.Description,
			&p.PriceCents,
			&p.TokenCredit,
			&p.IsSubscription,
			&p.StripePriceID,
			&p.AppleProductID,
			&p.GoogleProductID,
		); err != nil {
			return nil, fmt.Errorf("could not scan product: %w", err)
		}
		products = append(products, &p)
	}
	return products, nil
}

// GetProductByID fetches a single product
func (pr *postgresRepository) GetProductByID(ctx context.Context, productID string) (*domain.Product, error) {
	query := `
		SELECT 
			product_id, name, description, price_cents, 
			token_credit, is_subscription, stripe_price_id, 
			apple_product_id, google_product_id
		FROM products
		WHERE product_id = $1 
			OR apple_product_id = $1 
			OR google_product_id = $1
	`

	var p domain.Product
	err := pr.db.QueryRowContext(ctx, query, productID).Scan(
		&p.ProductID,
		&p.Name,
		&p.Description,
		&p.PriceCents,
		&p.TokenCredit,
		&p.IsSubscription,
		&p.StripePriceID,
		&p.AppleProductID,
		&p.GoogleProductID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("could not get product: %w", err)
	}
	return &p, nil
}

// CreateTransaction inserts a new row into payment_transactions.
func (pr *postgresRepository) CreateTransaction(ctx context.Context, tx *domain.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions
			(transaction_id, user_id, product_id, amount_cents, 
			 provider, provider_transaction_id, status, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := pr.db.ExecContext(ctx, query,
		tx.TransactionID,
		tx.UserID,
		tx.ProductID,
		tx.AmountCents,
		tx.Provider,
		tx.ProviderTransactionID,
		tx.Status,
		tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("could not insert transaction: %w", err)
	}
	return nil
}
