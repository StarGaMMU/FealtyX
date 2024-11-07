package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"bufio"

	"github.com/gorilla/mux"
)

type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

var (
	students  = make(map[int]Student)
	idCounter = 1

	studentMux sync.RWMutex
	//queue      = make(chan Request, 100) // queue with capacity of 100 (example) requests which holds the requests until workers are available to process them
)

/*type Request struct {
	w      http.ResponseWriter
	r      *http.Request
	action func(http.ResponseWriter, *http.Request)
}*/

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/students", CreateStudent).Methods("POST")
	router.HandleFunc("/students", GetAllStudents).Methods("GET")
	router.HandleFunc("/students/{id}", GetStudentByID).Methods("GET")
	router.HandleFunc("/students/{id}", UpdateStudentByID).Methods("PUT")
	router.HandleFunc("/students/{id}", DeleteStudentByID).Methods("DELETE")
	router.HandleFunc("/students/{id}/summary", GetStudentSummary).Methods("GET")
	router.HandleFunc("/", DefaultPage).Methods("GET")

	//worker pool
	/*or i := 0; i < 1; i++ {
		go worker()
	}*/

	fmt.Println("Server is starting on port 8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

// worker that handles requests from the queue
/*func worker() {
	for req := range queue {
		req.action(req.w, req.r)
	}
}

// function to enqueue requests
func enqueueRequest(action func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := Request{w: w, r: r, action: action}
		select {
		case queue <- req:
			// request added to queue successfully
		default:
			http.Error(w, "Server too busy, try again later", http.StatusServiceUnavailable) // Already busy
		}
	}
}*/

// Create Student
func CreateStudent(w http.ResponseWriter, r *http.Request) {
	fmt.Println("CreateStudent endpoint hit") // checking for func call aka hit
	w.Header().Set("Content-Type", "application/json")

	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		// Set the status code to 400 Bad Request for invalid payload
		w.WriteHeader(http.StatusBadRequest)
		// Custom error response
		response := map[string]interface{}{
			"status":  400,
			"message": "Invalid request payload",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Assigning a unique ID to the student and store it
	student.ID = idCounter
	idCounter++

	studentMux.Lock()
	students[student.ID] = student
	studentMux.Unlock()

	// Custom success response
	response := map[string]interface{}{
		"message": "Student created successfully",
		"student": student,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Get all students
func GetAllStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	studentList := make([]Student, 0, len(students))

	for _, student := range students {
		studentList = append(studentList, student)
	}

	// Encode the list of students as JSON
	if err := json.NewEncoder(w).Encode(studentList); err != nil {
		// If encoding fails, return a 500 error
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode students"})
		return
	}
}

// Get a student by ID
func GetStudentByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, _ := strconv.Atoi(params["id"])

	studentMux.RLock()
	student, exists := students[id]
	studentMux.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(student)
}

// Update a student by ID
func UpdateStudentByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, _ := strconv.Atoi(params["id"])

	studentMux.Lock()
	_, exists := students[id]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		studentMux.Unlock()
		return
	}
	var updatedStudent Student
	json.NewDecoder(r.Body).Decode(&updatedStudent)
	updatedStudent.ID = id
	students[id] = updatedStudent
	studentMux.Unlock()

	json.NewEncoder(w).Encode(updatedStudent)
}

// Delete a student by ID
func DeleteStudentByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, _ := strconv.Atoi(params["id"])

	studentMux.Lock()
	_, exists := students[id]
	if exists {
		delete(students, id)
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	studentMux.Unlock()
}

func queryOllama(model, prompt string) string {
	url := "http://localhost:11434/api/generate"

	// JSON payload with model and prompt
	payload := map[string]string{
		"model":  model,
		"prompt": prompt,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("failed to marshal payload: %v", err)
	}

	// Create a POST request to Ollama
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Sprintf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sending the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("request to Ollama failed: %v", err)
	}
	defer resp.Body.Close()

	// scanner to read response line by line
	scanner := bufio.NewScanner(resp.Body)
	var fullResponse string

	// Process each line of the response
	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Println("Raw Line:", line) // Debugging

		var ollamaResp OllamaResponse
		if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
			return fmt.Sprintf("failed to unmarshal response line: %v", err)
		}

		// Appending the response to the full response
		fullResponse += ollamaResp.Response

		// Break if `done` is true
		if ollamaResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("error reading response: %v", err)
	}

	return fullResponse
}

// Generating a summary of a student (by ID) using Ollama
func GetStudentSummary(w http.ResponseWriter, r *http.Request) {
	// Parse the student ID from the URL path
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	studentMux.RLock()
	student, exists := students[id]
	studentMux.RUnlock()

	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// prompt for Ollama with student details
	prompt := fmt.Sprintf("\"Summarize the profile of a student with the following details: Name: %s, Age: %d, Email: %s\"", student.Name, student.Age, student.Email)
	output := queryOllama("llama3", prompt)
	response := map[string]string{
		"summary": output,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func DefaultPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// introductory note
	introNote := "This is just a documentation of available API endpoints and how to use them."

	// API documentation structure
	apiInfo := map[string]interface{}{
		"POST": map[string]string{
			"/students": "Create a new student. Expected format: { 'Name': 'string', 'Age': 'int', 'Email': 'string' }",
		},
		"GET": map[string]string{
			"/students":              "Retrieve a list of all students.",
			"/students/{id}":         "Retrieve a student by ID.",
			"/students/{id}/summary": "Get a summary of a student by ID using Ollama AI.",
		},
		"PUT": map[string]string{
			"/students/{id}": "Update a student's information by ID. Expected format: { 'Name': 'string', 'Age': 'int', 'Email': 'string' }",
		},
		"DELETE": map[string]string{
			"/students/{id}": "Delete a student by ID.",
		},
	}

	// additional note
	apiNote := map[string]string{
		"NOTE": "**IMPORTANT:** To send requests for POST, UPDATE, DELETE you may use Postman, write a code script, or use sites like https://reqbin.com/post-online",
	}

	type DocumentationResponse struct {
		Introduction          string                 `json:"Introduction"`
		APIDocumentation      map[string]interface{} `json:"API Documentation"`
		AdditionalInformation map[string]string      `json:"Additional Information"`
	}

	response := DocumentationResponse{
		Introduction:          introNote,
		APIDocumentation:      apiInfo,
		AdditionalInformation: apiNote,
	}

	// Pretty-print
	prettyJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "Failed to format JSON", http.StatusInternalServerError)
		return
	}

	w.Write(prettyJSON)
}
