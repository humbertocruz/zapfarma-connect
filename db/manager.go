package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/nakagami/firebirdsql"
)

type DBManager struct {
	db     *sql.DB
	dbType string
}

func NewDBManager(dbType, host, port, user, password, dbname string) (*DBManager, error) {
	dsn := buildDSN(dbType, host, port, user, password, dbname)

	db, err := sql.Open(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBManager{db: db, dbType: dbType}, nil
}

func buildDSN(dbType, host, port, user, password, dbname string) string {
	switch dbType {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	case "sqlserver":
		return fmt.Sprintf("server=%s;port=%s;user id=%s;password=%s;database=%s", host, port, user, password, dbname)
	case "firebird":
		return fmt.Sprintf("%s:%s@%s:%s/%s", user, password, host, port, dbname)
	case "sqlite":
		return dbname
	default:
		return ""
	}
}

func (m *DBManager) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
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

	return results, nil
}

func (m *DBManager) Exec(query string, args ...interface{}) (int64, error) {
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *DBManager) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

func (m *DBManager) Ping() error {
	return m.db.Ping()
}
