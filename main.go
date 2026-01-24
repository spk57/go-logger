// go-logger - HTTP API server for logging data entries from Arduino/remote devices
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	logFile     = "logger.csv"
	defaultPort = "8765"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID          int       `json:"id"`
	Transaction string    `json:"transaction"`
	Datetime    time.Time `json:"datetime"`
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
}

// Logger handles all log operations with thread safety
type Logger struct {
	mu       sync.RWMutex
	filePath string
}

// NewLogger creates a new Logger instance and initializes the CSV file
func NewLogger(filePath string) (*Logger, error) {
	l := &Logger{filePath: filePath}
	if err := l.initFile(); err != nil {
		return nil, err
	}
	return l, nil
}

// initFile creates the CSV file with headers if it doesn't exist
func (l *Logger) initFile() error {
	if _, err := os.Stat(l.filePath); os.IsNotExist(err) {
		file, err := os.Create(l.filePath)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		headers := []string{"id", "transaction", "datetime", "name", "value", "source", "created_at"}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write headers: %w", err)
		}
		writer.Flush()
	}
	return nil
}

// readAllEntries reads all entries from the CSV file
func (l *Logger) readAllEntries() ([]LogEntry, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	for i, record := range records {
		if i == 0 { // skip header
			continue
		}
		if len(record) < 7 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		datetime, _ := time.Parse(time.RFC3339, record[2])
		createdAt, _ := time.Parse(time.RFC3339, record[6])

		entries = append(entries, LogEntry{
			ID:          id,
			Transaction: record[1],
			Datetime:    datetime,
			Name:        record[3],
			Value:       record[4],
			Source:      record[5],
			CreatedAt:   createdAt,
		})
	}
	return entries, nil
}

// AddEntry adds a new log entry
func (l *Logger) AddEntry(datetime time.Time, transaction, name, value, source string) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := l.readAllEntries()
	if err != nil {
		return 0, err
	}

	nextID := 1
	if len(entries) > 0 {
		maxID := 0
		for _, e := range entries {
			if e.ID > maxID {
				maxID = e.ID
			}
		}
		nextID = maxID + 1
	}

	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	record := []string{
		strconv.Itoa(nextID),
		transaction,
		datetime.Format(time.RFC3339),
		name,
		value,
		source,
		time.Now().Format(time.RFC3339),
	}
	if err := writer.Write(record); err != nil {
		return 0, err
	}
	writer.Flush()

	return nextID, nil
}

// GetEntries retrieves log entries with optional filtering
func (l *Logger) GetEntries(limit, offset int, source, name string) ([]LogEntry, int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries, err := l.readAllEntries()
	if err != nil {
		return nil, 0, err
	}

	// Apply filters
	var filtered []LogEntry
	for _, e := range entries {
		if source != "" && e.Source != source {
			continue
		}
		if name != "" && e.Name != name {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)

	// Apply pagination
	if offset >= total {
		return []LogEntry{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total, nil
}

// ClearEntries removes all log entries
func (l *Logger) ClearEntries() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Create(l.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	headers := []string{"id", "transaction", "datetime", "name", "value", "source", "created_at"}
	if err := writer.Write(headers); err != nil {
		return err
	}
	writer.Flush()
	return nil
}

// GetLocationForSource retrieves the most recent location for a given source
// Returns the location value and true if found, empty string and false otherwise
func (l *Logger) GetLocationForSource(source string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries, err := l.readAllEntries()
	if err != nil {
		return "", false
	}

	// Find the most recent set_location entry for this source
	var mostRecent *LogEntry
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Source == source && e.Transaction == "set_location" && e.Name == "location" {
			if mostRecent == nil || e.CreatedAt.After(mostRecent.CreatedAt) {
				mostRecent = &e
			}
		}
	}

	if mostRecent != nil {
		return mostRecent.Value, true
	}
	return "", false
}

// GetStats returns statistics about the log entries
func (l *Logger) GetStats() (map[string]interface{}, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries, err := l.readAllEntries()
	if err != nil {
		return nil, err
	}

	sourcesMap := make(map[string]bool)
	namesMap := make(map[string]bool)

	for _, e := range entries {
		sourcesMap[e.Source] = true
		namesMap[e.Name] = true
	}

	sources := make([]string, 0, len(sourcesMap))
	for s := range sourcesMap {
		sources = append(sources, s)
	}

	names := make([]string, 0, len(namesMap))
	for n := range namesMap {
		names = append(names, n)
	}

	return map[string]interface{}{
		"success":        true,
		"total_entries":  len(entries),
		"unique_sources": len(sources),
		"unique_names":   len(names),
		"sources":        sources,
		"names":          names,
	}, nil
}

// API Server
type Server struct {
	logger *Logger
}

func NewServer(logger *Logger) *Server {
	return &Server{logger: logger}
}

// AddLogRequest is the request body for adding a log entry
type AddLogRequest struct {
	Datetime    string      `json:"datetime"`    // RFC3339 format, or empty for current time
	Transaction string      `json:"transaction"` // Transaction identifier
	Name        string      `json:"name"`
	Value       interface{} `json:"value"` // accepts string, number, or other JSON types
	Source      string      `json:"source"`
}

// enableCORS adds CORS headers to allow Arduino/ESP requests
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (s *Server) handleLog(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.addLogEntry(w, r)
	case http.MethodGet:
		s.getLogEntries(w, r)
	case http.MethodDelete:
		s.clearLogEntries(w, r)
	default:
		http.Error(w, `{"success":false,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) addLogEntry(w http.ResponseWriter, r *http.Request) {
	var req AddLogRequest

	// Support both JSON body and URL query parameters (for simple Arduino GET requests)
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid JSON: " + err.Error(),
			})
			return
		}
	} else {
		// Parse from query params for simpler Arduino integration
		req.Name = r.URL.Query().Get("name")
		req.Value = r.URL.Query().Get("value")
		req.Source = r.URL.Query().Get("source")
		req.Datetime = r.URL.Query().Get("datetime")
		req.Transaction = r.URL.Query().Get("transaction")
	}

	// Validate required fields
	if req.Datetime == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "missing required field: datetime",
		})
		return
	}

	if req.Name == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "missing required field: name",
		})
		return
	}

	// Parse datetime
	var dt time.Time
	var err error
	dt, err = time.Parse(time.RFC3339, req.Datetime)
	if err != nil {
		// Try simpler format
		dt, err = time.Parse("2006-01-02T15:04:05", req.Datetime)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "invalid datetime format, use RFC3339 or YYYY-MM-DDTHH:MM:SS",
			})
			return
		}
	}

	// Convert value to string for storage
	var valueStr string
	switch v := req.Value.(type) {
	case string:
		valueStr = v
	case float64:
		valueStr = strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		valueStr = strconv.Itoa(v)
	case nil:
		valueStr = ""
	default:
		valueStr = fmt.Sprintf("%v", v)
	}

	// Check for set_location transaction for this source
	// If found, use the location value instead of the reported transaction
	location, hasLocation := s.logger.GetLocationForSource(req.Source)
	effectiveTransaction := req.Transaction
	if hasLocation && req.Transaction != "set_location" {
		// Use the location value instead of the reported transaction from the device
		effectiveTransaction = location
	}

	id, err := s.logger.AddEntry(dt, effectiveTransaction, req.Name, valueStr, req.Source)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to add entry: " + err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Log entry created successfully",
		"id":      id,
	}

	// Include original transaction and location info if location was used
	if hasLocation && req.Transaction != "set_location" {
		response["location"] = location
		response["original_transaction"] = req.Transaction
		response["transaction"] = effectiveTransaction
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) getLogEntries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 100
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	source := query.Get("source")
	name := query.Get("name")

	entries, total, err := s.logger.GetEntries(limit, offset, source, name)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to get entries: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (s *Server) clearLogEntries(w http.ResponseWriter, r *http.Request) {
	if err := s.logger.ClearEntries(); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to clear entries: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "All log entries cleared",
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, `{"success":false,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.logger.GetStats()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to get stats: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// handleQuickLog provides a simple endpoint for Arduino GET requests
// Example: GET /quick?name=temperature&value=23.5&source=arduino-1
func (s *Server) handleQuickLog(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	query := r.URL.Query()
	name := query.Get("name")
	value := query.Get("value")
	source := query.Get("source")
	transaction := query.Get("transaction")

	if name == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "name parameter is required",
		})
		return
	}

	// Check for set_location transaction for this source
	// If found, use the location value instead of the reported transaction
	location, hasLocation := s.logger.GetLocationForSource(source)
	effectiveTransaction := transaction
	if hasLocation && transaction != "set_location" {
		// Use the location value instead of the reported transaction from the device
		effectiveTransaction = location
	}

	id, err := s.logger.AddEntry(time.Now(), effectiveTransaction, name, value, source)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to add entry: " + err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Log entry created successfully",
		"id":      id,
	}

	// Include original transaction and location info if location was used
	if hasLocation && transaction != "set_location" {
		response["location"] = location
		response["original_transaction"] = transaction
		response["transaction"] = effectiveTransaction
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	logFilePath := os.Getenv("LOG_FILE")
	if logFilePath == "" {
		logFilePath = logFile
	}

	logger, err := NewLogger(logFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	server := NewServer(logger)

	http.HandleFunc("/log", server.handleLog)
	http.HandleFunc("/logs", server.handleLog)
	http.HandleFunc("/api/logger", server.handleLog)
	http.HandleFunc("/api/logger/stats", server.handleStats)
	http.HandleFunc("/stats", server.handleStats)
	http.HandleFunc("/quick", server.handleQuickLog)
	http.HandleFunc("/health", server.handleHealth)

	// Get local IP addresses for remote access info
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0" // Bind to all interfaces by default
	}

	fmt.Printf("🚀 Go Logger API Server starting on %s:%s\n", host, port)
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/logger       - Add a log entry (JSON body)")
	fmt.Println("  GET    /api/logger       - Get log entries (with ?limit, ?offset, ?source, ?name filters)")
	fmt.Println("  DELETE /api/logger       - Clear all log entries")
	fmt.Println("  GET    /api/logger/stats - Get log statistics")
	fmt.Println("  POST   /log              - Add a log entry (JSON body or query params)")
	fmt.Println("  GET    /log              - Get log entries (with ?limit, ?offset, ?source, ?name filters)")
	fmt.Println("  DELETE /log              - Clear all log entries")
	fmt.Println("  GET    /stats            - Get log statistics")
	fmt.Println("  GET    /quick            - Quick log entry (query params: name, value, source)")
	fmt.Println("  GET    /health           - Health check")

	// Display network information for remote access
	if host == "0.0.0.0" || host == "" {
		fmt.Println("\n📡 Server is listening on all network interfaces")
		fmt.Println("   Local access:  http://localhost:" + port)
		fmt.Println("   Remote access:  http://<your-ip>:" + port)
		fmt.Println("   Use 'ip addr' or 'hostname -I' to find your IP address")
	} else {
		fmt.Printf("\n📡 Server is listening on %s:%s\n", host, port)
	}
	fmt.Printf("\nArduino example: GET http://<server-ip>:%s/quick?name=temp&value=25.5&source=arduino-1\n\n", port)

	addr := host + ":" + port
	log.Fatal(http.ListenAndServe(addr, nil))
}
