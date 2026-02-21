package main

import (
	"db-bridge/config"
	"db-bridge/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

var cfg *config.Config
var dbManager *db.DBManager
var token string

func getBaseURL() string {
	if url := os.Getenv("ZAPFARMA_URL"); url != "" {
		return url
	}
	return "https://zapfarma.vercel.app"
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		runConsole()
		return
	}

	startApp()
}

func startApp() {
	cfg = config.Load()

	// Se tem token, verifica se é válido
	if cfg.Token != "" {
		valid, err := verifyToken(cfg.Token)
		if err == nil && valid {
			startPolling()
			openBrowser(getBaseURL() + "/dashboard?token=" + cfg.Token)
			log.Println("Polling started. Press Ctrl+C to stop.")
			// Espera para sempre
			select {}
		}
		log.Printf("Token inválido, pedindo novo login")
	}

	loginAndConfigure()
}

func loginAndConfigure() {
	log.Printf("Opening login page: %s/connect", getBaseURL())
	openBrowser(getBaseURL() + "/connect")

	token = waitForToken()
	if token == "" {
		log.Println("No token received, exiting")
		os.Exit(1)
	}

	log.Printf("Token received, opening configuration page")
	openBrowser(getBaseURL() + "/configure-client?token=" + token)

	waitForConfig()
}

func waitForToken() string {
	for i := 0; i < 300; i++ {
		time.Sleep(1 * time.Second)

		token, err := fetchToken()
		if err == nil && token != "" {
			log.Println("Token received!")
			saveToken(token)
			return token
		}

		if i%10 == 0 {
			log.Printf("Waiting for login... (%ds)", i)
		}
	}
	return ""
}

func fetchToken() (string, error) {
	if cfg.Token == "" {
		return "", fmt.Errorf("no token")
	}

	resp, err := http.Get(getBaseURL() + "/api/auth/verify?token=" + cfg.Token)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("invalid token")
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Valid {
		return cfg.Token, nil
	}

	return "", fmt.Errorf("token not valid")
}

func verifyToken(t string) (bool, error) {
	resp, err := http.Get(getBaseURL() + "/api/auth/verify?token=" + t)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Valid, nil
}

func saveToken(t string) {
	cfg.Token = t
	lines := []string{}

	if data, err := os.ReadFile(".env"); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "TOKEN=") {
			lines[i] = "TOKEN=" + t
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, "TOKEN="+t)
	}

	os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0644)
}

func waitForConfig() {
	for i := 0; i < 300; i++ {
		time.Sleep(1 * time.Second)

		configData, err := fetchConfig()
		if err == nil && configData != nil {
			log.Println("Configuration received!")
			saveConfig(configData)
			log.Println("Configuration saved! Closing...")
			os.Exit(0)
		}

		if i%10 == 0 {
			log.Printf("Waiting for configuration... (%ds)", i)
		}
	}
}

func fetchConfig() (*config.Config, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("no token")
	}

	resp, err := http.Get(getBaseURL() + "/api/config/get?token=" + cfg.Token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get config")
	}

	var result struct {
		Success bool `json:"success"`
		Config  *struct {
			ClientID     string `json:"clientId"`
			System       string `json:"system"`
			DBType       string `json:"dbType"`
			DBHost       string `json:"dbHost"`
			DBPort       string `json:"dbPort"`
			DBUser       string `json:"dbUser"`
			DBPassword   string `json:"dbPassword"`
			DBName       string `json:"dbName"`
			ServerHost   string `json:"serverHost"`
			ServerPort   string `json:"serverPort"`
			PollInterval int    `json:"pollInterval"`
		} `json:"config"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success || result.Config == nil {
		return nil, fmt.Errorf("no config")
	}

	c := result.Config
	return &config.Config{
		ClientID:     c.ClientID,
		System:       c.System,
		DBType:       c.DBType,
		DBHost:       c.DBHost,
		DBPort:       c.DBPort,
		DBUser:       c.DBUser,
		DBPassword:   c.DBPassword,
		DBName:       c.DBName,
		ServerHost:   c.ServerHost,
		ServerPort:   c.ServerPort,
		PollInterval: c.PollInterval,
		ProductTable: getProductTable(c.System),
		ProductCode:  getProductCode(c.System),
		ProductName:  getProductName(c.System),
		ProductQty:   getProductQty(c.System),
		ProductPrice: getProductPrice(c.System),
		SearchQuery:  getSearchQuery(c.System),
	}, nil
}

func saveConfig(c *config.Config) {
	lines := []string{
		"# ZapFarma Connect Configuration",
		fmt.Sprintf("TOKEN=%s", token),
		"",
		fmt.Sprintf("SYSTEM=%s", c.System),
		fmt.Sprintf("SERVER_HOST=%s", c.ServerHost),
		fmt.Sprintf("SERVER_PORT=%s", c.ServerPort),
		fmt.Sprintf("CLIENT_ID=%s", c.ClientID),
		fmt.Sprintf("POLL_INTERVAL=%d", c.PollInterval),
		"",
		fmt.Sprintf("DB_TYPE=%s", c.DBType),
		fmt.Sprintf("DB_HOST=%s", c.DBHost),
		fmt.Sprintf("DB_PORT=%s", c.DBPort),
		fmt.Sprintf("DB_USER=%s", c.DBUser),
		fmt.Sprintf("DB_PASSWORD=%s", c.DBPassword),
		fmt.Sprintf("DB_NAME=%s", c.DBName),
		"",
		fmt.Sprintf("PRODUCT_TABLE=%s", c.ProductTable),
		fmt.Sprintf("PRODUCT_CODE=%s", c.ProductCode),
		fmt.Sprintf("PRODUCT_NAME=%s", c.ProductName),
		fmt.Sprintf("PRODUCT_QTY=%s", c.ProductQty),
		fmt.Sprintf("PRODUCT_PRICE=%s", c.ProductPrice),
		fmt.Sprintf("SEARCH_QUERY=%s", c.SearchQuery),
	}
	os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0644)
}

func getProductTable(system string) string {
	if system == "digifarma" {
		return "PRODUTO"
	}
	return "Produto"
}

func getProductCode(system string) string {
	if system == "digifarma" {
		return "CODIGO"
	}
	return "Codigo"
}

func getProductName(system string) string {
	if system == "digifarma" {
		return "DESCRICAO"
	}
	return "Descricao"
}

func getProductQty(system string) string {
	if system == "digifarma" {
		return "ESTOQUE_ATUAL"
	}
	return "EstoqueAtual"
}

func getProductPrice(system string) string {
	if system == "digifarma" {
		return "PRECO_VAREJO"
	}
	return "PrecoVarejo"
}

func getSearchQuery(system string) string {
	if system == "digifarma" {
		return "UPPER(DESCRICAO) CONTAINING '{term}'"
	}
	return "UPPER(Descricao) LIKE '%{term}%'"
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	cmd.Start()
}

func startPolling() {
	var err error
	dbManager, err = db.NewDBManager(cfg.DBType, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Printf("DB connection error: %v", err)
		return
	}

	go pollLoop()
}

func pollLoop() {
	for {
		result := searchProducts()
		sendResult(result)
		time.Sleep(time.Duration(cfg.PollInterval) * time.Second)
	}
}

func searchProducts() SearchResult {
	searchTerm, err := fetchSearchTerm()
	if err != nil {
		return SearchResult{Success: false, Error: err.Error()}
	}

	if searchTerm == "" {
		return SearchResult{Success: true, Products: []Product{}}
	}

	query := buildProductQuery(searchTerm)
	rows, err := dbManager.Query(query)
	if err != nil {
		return SearchResult{Success: false, Error: err.Error()}
	}

	products := parseProducts(rows)
	return SearchResult{Success: true, Products: products}
}

func buildProductQuery(searchTerm string) string {
	term := strings.ToUpper(searchTerm)
	searchCond := strings.ReplaceAll(cfg.SearchQuery, "{term}", term)
	searchCond = strings.ReplaceAll(searchCond, "{name}", cfg.ProductName)

	if cfg.DBType == "sqlserver" {
		return fmt.Sprintf(`SELECT TOP 20 %s as codigo, %s as nome, %s as quantidade, %s as preco FROM %s WHERE %s ORDER BY %s`,
			cfg.ProductCode, cfg.ProductName, cfg.ProductQty, cfg.ProductPrice, cfg.ProductTable, searchCond, cfg.ProductName)
	}

	return fmt.Sprintf(`SELECT %s as codigo, %s as nome, %s as quantidade, %s as preco FROM %s WHERE %s ORDER BY %s LIMIT 20`,
		cfg.ProductCode, cfg.ProductName, cfg.ProductQty, cfg.ProductPrice, cfg.ProductTable, searchCond, cfg.ProductName)
}

func fetchSearchTerm() (string, error) {
	url := fmt.Sprintf("%s/api/bridge/search?clientId=%s&token=%s", getBaseURL(), cfg.ClientID, token)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		return "", nil
	}

	var result struct {
		SearchTerm string `json:"searchTerm"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SearchTerm, nil
}

func sendResult(result SearchResult) error {
	url := fmt.Sprintf("%s/api/bridge/result?clientId=%s&token=%s", getBaseURL(), cfg.ClientID, token)
	body, _ := json.Marshal(result)
	http.Post(url, "application/json", strings.NewReader(string(body)))
	return nil
}

type SearchResult struct {
	Success  bool      `json:"success"`
	Products []Product `json:"products,omitempty"`
	Error    string    `json:"error,omitempty"`
}

type Product struct {
	Codigo     string  `json:"codigo"`
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Preco      float64 `json:"preco"`
}

func parseProducts(rows []map[string]interface{}) []Product {
	products := make([]Product, 0)
	for _, row := range rows {
		p := Product{}
		if v, ok := row["codigo"]; ok {
			p.Codigo = fmt.Sprintf("%v", v)
		}
		if v, ok := row["nome"]; ok {
			p.Nome = fmt.Sprintf("%v", v)
		}
		if v, ok := row["quantidade"]; ok {
			switch n := v.(type) {
			case float64:
				p.Quantidade = int(n)
			case int64:
				p.Quantidade = int(n)
			}
		}
		if v, ok := row["preco"]; ok {
			switch n := v.(type) {
			case float64:
				p.Preco = n
			}
		}
		products = append(products, p)
	}
	return products
}

func runConsole() {
	cfg = config.Load()

	log.Printf("Connecting to %s database...", cfg.DBType)
	dbManager, err := db.NewDBManager(cfg.DBType, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer dbManager.Close()
	log.Println("Database connected")
	log.Printf("Polling server at %s every %d seconds", cfg.ServerHost, cfg.PollInterval)

	pollLoop()
}
