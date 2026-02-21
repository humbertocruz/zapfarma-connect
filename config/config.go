package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Token        string
	ServerPort   string
	ServerHost   string
	ClientID     string
	PollInterval int
	DebugTable   string
	System       string
	DBType       string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	ProductTable string
	ProductCode  string
	ProductName  string
	ProductQty   string
	ProductPrice string
	SearchQuery  string
	AuthUsername string
	AuthPassword string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() *Config {
	system := getEnv("SYSTEM", "digifarma")

	productTable, productCode, productName, productQty, productPrice, searchQuery := getSystemConfig(system)

	return &Config{
		Token:        getEnv("TOKEN", ""),
		ServerPort:   getEnv("SERVER_PORT", "3000"),
		ServerHost:   getEnv("SERVER_HOST", "localhost"),
		ClientID:     getEnv("CLIENT_ID", "cliente-001"),
		PollInterval: getInt("POLL_INTERVAL", 10),
		DebugTable:   getEnv("DEBUG_TABLE", ""),
		System:       system,
		DBType:       getEnv("DB_TYPE", "firebird"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "3050"),
		DBUser:       getEnv("DB_USER", "SYSDBA"),
		DBPassword:   getEnv("DB_PASSWORD", "masterkey"),
		DBName:       getEnv("DB_NAME", "C:\\Digifarma6\\Digifarma6.FDB"),

		ProductTable: getEnv("PRODUCT_TABLE", productTable),
		ProductCode:  getEnv("PRODUCT_CODE", productCode),
		ProductName:  getEnv("PRODUCT_NAME", productName),
		ProductQty:   getEnv("PRODUCT_QTY", productQty),
		ProductPrice: getEnv("PRODUCT_PRICE", productPrice),
		SearchQuery:  getEnv("SEARCH_QUERY", searchQuery),

		AuthUsername: getEnv("AUTH_USERNAME", "admin"),
		AuthPassword: getEnv("AUTH_PASSWORD", "password"),
		ReadTimeout:  getDuration("READ_TIMEOUT", 30),
		WriteTimeout: getDuration("WRITE_TIMEOUT", 30),
	}
}

func getSystemConfig(system string) (table, code, name, qty, price, query string) {
	switch system {
	case "digifarma":
		return "PRODUTO", "CODIGO", "DESCRICAO", "ESTOQUE_ATUAL", "PRECO_VAREJO", "UPPER({name}) CONTAINING '{term}'"
	case "inovafarma":
		return "Produto", "Codigo", "Descricao", "EstoqueAtual", "PrecoVarejo", "UPPER(Descricao) LIKE '%{term}%'"
	default:
		return "PRODUTO", "CODIGO", "DESCRICAO", "ESTOQUE", "PRECO", "UPPER(NOME) LIKE '%{term}%'"
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return time.Duration(v) * time.Second
		}
	}
	return time.Duration(defaultValue) * time.Second
}
