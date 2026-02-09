// go-logger - HTTP API server for logging data entries from Arduino/remote devices
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
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

var (
	debugMode = false
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID          int       `json:"id"`
	Transaction string    `json:"transaction"`
	Datetime    time.Time `json:"datetime"`
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Source      string    `json:"source"`
	Location    string    `json:"location"`
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
// If an old format file exists (7 fields), it removes it and creates a new one
func (l *Logger) initFile() error {
	_, err := os.Stat(l.filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, create new one
		return l.createNewFile()
	}
	if err != nil {
		return fmt.Errorf("failed to check log file: %w", err)
	}

	// File exists, check if it's old format (7 fields) or new format (8 fields)
	file, err := os.Open(l.filePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	file.Close() // Close before potentially removing file
	if err != nil {
		// If we can't read headers, assume old format and recreate
		os.Remove(l.filePath)
		return l.createNewFile()
	}

	// Check if it's old format (7 fields) or new format (8 fields with location)
	if len(headers) < 8 || (len(headers) >= 7 && headers[6] != "location") {
		// Old format detected, remove and create new
		os.Remove(l.filePath)
		return l.createNewFile()
	}

	// New format file exists, nothing to do
	return nil
}

// createNewFile creates a new CSV file with the correct headers
func (l *Logger) createNewFile() error {
	file, err := os.Create(l.filePath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	headers := []string{"id", "transaction", "datetime", "name", "value", "source", "location", "created_at"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	writer.Flush()
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
		// New format requires 8 fields: id, transaction, datetime, name, value, source, location, created_at
		if len(record) < 8 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		datetime, _ := time.Parse(time.RFC3339, record[2])
		createdAt, _ := time.Parse(time.RFC3339, record[7])

		entries = append(entries, LogEntry{
			ID:          id,
			Transaction: record[1],
			Datetime:    datetime,
			Name:        record[3],
			Value:       record[4],
			Source:      record[5],
			Location:    record[6],
			CreatedAt:   createdAt,
		})
	}
	return entries, nil
}

// AddEntry adds a new log entry
func (l *Logger) AddEntry(datetime time.Time, transaction, name, value, source, location string) (int, error) {
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
		location,
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
	headers := []string{"id", "transaction", "datetime", "name", "value", "source", "location", "created_at"}
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
	Note        string      `json:"note"` // Note content for "note" transactions
}

// enableCORS adds CORS headers to allow Arduino/ESP requests
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// debugLog logs connection requests when debug mode is enabled
func debugLog(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if debugMode {
			timestamp := time.Now().Format("2006-01-02 15:04:05")
			fmt.Printf("[DEBUG] %s - %s %s from %s\n", timestamp, r.Method, r.URL.Path, r.RemoteAddr)
			if r.URL.RawQuery != "" {
				fmt.Printf("[DEBUG]   Query: %s\n", r.URL.RawQuery)
			}
			if r.Method == "POST" || r.Method == "PUT" {
				contentType := r.Header.Get("Content-Type")
				fmt.Printf("[DEBUG]   Content-Type: %s\n", contentType)
			}
		}
		handler(w, r)
	}
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
		req.Note = r.URL.Query().Get("note")
	}

	// Validate required fields
	if req.Datetime == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "missing required field: datetime",
		})
		return
	}

	// Handle note transaction - note field is required, name is set to "note"
	if req.Transaction == "note" {
		if req.Note == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "missing required field: note",
			})
			return
		}
		// Set name to "note" and value to the note content
		req.Name = "note"
		req.Value = req.Note
	} else if req.Name == "" {
		// For non-note transactions, name is required
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

	// Determine location for this entry
	var location string
	if req.Transaction == "set_location" && req.Name == "location" {
		// For set_location transactions, use the value as the location
		location = valueStr
	} else {
		// For other transactions, check if there's a set_location for this source
		location, _ = s.logger.GetLocationForSource(req.Source)
	}

	id, err := s.logger.AddEntry(dt, req.Transaction, req.Name, valueStr, req.Source, location)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to add entry: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Log entry created successfully",
		"id":      id,
	})
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
	note := query.Get("note")

	// Handle note transaction
	if transaction == "note" {
		if note == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "note parameter is required for note transactions",
			})
			return
		}
		name = "note"
		value = note
	} else if name == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "name parameter is required",
		})
		return
	}

	// Determine location for this entry
	var location string
	if transaction == "set_location" && name == "location" {
		// For set_location transactions, use the value as the location
		location = value
	} else {
		// For other transactions, check if there's a set_location for this source
		location, _ = s.logger.GetLocationForSource(source)
	}

	id, err := s.logger.AddEntry(time.Now(), transaction, name, value, source, location)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to add entry: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Log entry created successfully",
		"id":      id,
	})
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
	// Parse command line flags
	flag.BoolVar(&debugMode, "d", false, "Enable debug mode (log connection requests)")
	flag.Parse()

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

	// Register handlers with debug logging wrapper
	http.HandleFunc("/log", debugLog(server.handleLog))
	http.HandleFunc("/logs", debugLog(server.handleLog))
	http.HandleFunc("/api/logger", debugLog(server.handleLog))
	http.HandleFunc("/api/logger/stats", debugLog(server.handleStats))
	http.HandleFunc("/stats", debugLog(server.handleStats))
	http.HandleFunc("/quick", debugLog(server.handleQuickLog))
	http.HandleFunc("/health", debugLog(server.handleHealth))

	// Get local IP addresses for remote access info
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0" // Bind to all interfaces by default
	}

	fmt.Printf("🚀 Go Logger API Server starting on %s:%s\n", host, port)
	if debugMode {
		fmt.Println("🔍 Debug mode: ENABLED (connection requests will be logged)")
	}
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
	fmt.Println("\nUsage: ./main [-d] to enable debug mode")

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
