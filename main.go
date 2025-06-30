package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	emailRegex          = regexp.MustCompile(`[\w\.-]+@[\w\.-]+\.[a-zA-Z]{2,}`)
	phoneRegex          = regexp.MustCompile(`(\+62|0)[0-9\s\-]{9,}`)
	yearRangeRegex      = regexp.MustCompile(`(19|20)\d{2}\s*[-–]\s*(19|20)\d{2}`)
	linkedinRegex       = regexp.MustCompile(`(?i)https?://(www\.)?linkedin\.com/in/[a-zA-Z0-9\-_%]+`)
	githubRegex         = regexp.MustCompile(`(?i)https?://(www\.)?github\.com/[a-zA-Z0-9\-_%]+`)
	urlRegex            = regexp.MustCompile(`https?://[^\s]+`)
	knownSkills         = []string{"Go", "Golang", "Java", "Python", "Docker", "PostgreSQL"}
	educationKeywords   = []string{"bachelor", "master", "diploma", "sma", "smk", "sarjana"}
	institutionKeywords = []string{"university", "institute", "college", "school", "politeknik"}
	positionKeywords    = []string{"engineer", "developer", "manager", "intern", "analyst", "officer", "staff"}
	companyKeywords     = []string{"corp", "company", "studio", "pt", "co.", "group", "llc", "indonesia"}
)

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	http.HandleFunc("/parse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "File upload error", http.StatusBadRequest)
			return
		}
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		var text string
		switch ext {
		case ".pdf":
			text, err = extractTextFromPDF(file)
		case ".docx":
			text, err = extractTextFromDocx(file)
		default:
			http.Error(w, "Unsupported file type", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "File parse error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resume := parseResume(text)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resume)
	})

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
