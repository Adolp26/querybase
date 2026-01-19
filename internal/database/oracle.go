package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/adolp26/querybase/internal/models"
	_ "github.com/sijms/go-ora/v2"
)

type OracleDataSource struct {
	db     *sql.DB
	config models.OracleConfig
}

func NewOracleDataSource(cfg models.OracleConfig) (*OracleDataSource, error) {
	dsn := fmt.Sprintf(
		"oracle://%s:%s@%s:%s/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Service,
	)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão Oracle: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar no Oracle: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(300 * time.Second)

	return &OracleDataSource{
		db:     db,
		config: cfg,
	}, nil
}

func (o *OracleDataSource) Query(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := o.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}

		results = append(results, row)
	}

	return results, rows.Err()
}

func (o *OracleDataSource) Ping(ctx context.Context) error {
	return o.db.PingContext(ctx)
}

func (o *OracleDataSource) Close() error {
	return o.db.Close()
}
