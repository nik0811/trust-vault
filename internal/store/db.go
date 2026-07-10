package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*sqlx.DB
}

func NewDB(url string) (*DB, error) {
	if url == "" {
		url = "postgres://trustvault:trustvault@localhost:5432/trustvault?sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &DB{db}, nil
}

func RunMigrations(db *DB, direction string, steps int) error {
	// Simple migration runner - reads SQL files from embedded FS
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if direction == "up" && !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		if direction == "down" && !strings.HasSuffix(entry.Name(), ".down.sql") {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", entry.Name(), err)
		}
	}
	return nil
}

type JSONB map[string]any

type TenantScoped struct {
	TenantID  string    `db:"tenant_id" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Repository[T any] struct {
	db    *DB
	table string
}

func NewRepo[T any](db *DB, table string) *Repository[T] {
	return &Repository[T]{db: db, table: table}
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	var cols, placeholders []string
	var args []any
	idx := 1

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		if dbTag == "id" {
			idField := v.Field(i)
			if idField.String() == "" {
				idField.SetString(uuid.New().String())
			}
		}
		if dbTag == "created_at" || dbTag == "updated_at" {
			v.Field(i).Set(reflect.ValueOf(time.Now()))
		}
		cols = append(cols, dbTag)
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		args = append(args, v.Field(i).Interface())
		idx++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository[T]) FindByID(ctx context.Context, tenantID, id string) (*T, error) {
	var entity T
	query := fmt.Sprintf("SELECT * FROM %s WHERE tenant_id = $1 AND id = $2", r.table)
	if err := r.db.GetContext(ctx, &entity, query, tenantID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *Repository[T]) List(ctx context.Context, tenantID string, opts ListOpts) ([]T, error) {
	var entities []T
	query := fmt.Sprintf("SELECT * FROM %s WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		r.table)
	if err := r.db.SelectContext(ctx, &entities, query, tenantID, opts.Limit, opts.Offset); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	var sets []string
	var args []any
	var id, tenantID string
	idx := 1

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" || dbTag == "created_at" {
			continue
		}
		if dbTag == "id" {
			id = v.Field(i).String()
			continue
		}
		if dbTag == "tenant_id" {
			tenantID = v.Field(i).String()
			continue
		}
		if dbTag == "updated_at" {
			v.Field(i).Set(reflect.ValueOf(time.Now()))
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", dbTag, idx))
		args = append(args, v.Field(i).Interface())
		idx++
	}

	args = append(args, tenantID, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE tenant_id = $%d AND id = $%d",
		r.table, strings.Join(sets, ", "), idx, idx+1)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository[T]) Delete(ctx context.Context, tenantID, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1 AND id = $2", r.table)
	_, err := r.db.ExecContext(ctx, query, tenantID, id)
	return err
}

type ListOpts struct {
	Limit  int
	Offset int
	Sort   string
	Filter map[string]any
}

func DefaultListOpts() ListOpts {
	return ListOpts{Limit: 50, Offset: 0}
}
