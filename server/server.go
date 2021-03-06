package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
)

// SearchRequest describes an incoming search request.
type SearchRequest struct {
	Title       string `schema:"title"`
	Description string `schema:"description"`
	Auth        string `schema:"auth"`
	HTTPS       string `schema:"https"`
	Category    string `schema:"category"`
}

// Entries contains an array of API entries, and a count representing the length of that array.
type Entries struct {
	Count   int     `json:"count"`
	Entries []Entry `json:"entries"`
}

// Entry describes a single API reference.
type Entry struct {
	API         string
	Description string
	Auth        string
	HTTPS       bool
	Link        string
	Category    string
}

// checkEntryMatches checks if the given entry matches the given request's parameters.
// it returns true if the entry matches, and returns false otherwise.
func checkEntryMatches(entry Entry, request *SearchRequest) bool {
	if strings.Contains(strings.ToLower(entry.API), strings.ToLower(request.Title)) &&
		strings.Contains(strings.ToLower(entry.Description), strings.ToLower(request.Description)) &&
		strings.Contains(strings.ToLower(entry.Auth), strings.ToLower(request.Auth)) &&
		strings.Contains(strings.ToLower(entry.Category), strings.ToLower(request.Category)) {
		if request.HTTPS == "" {
			return true
		} else {
			if value, err := strconv.ParseBool(request.HTTPS); err == nil {
				if entry.HTTPS == value {
					return true
				}
			}
		}
	}
	return false
}

func main() {
	// Open API entry file as a reader.
	file, err := os.OpenFile("../json/entries.min.json", os.O_RDONLY, 0644)
	if err != nil {
		panic("failed to open entries.min.json: " + err.Error())
	}

	// Decode file's contents into an Entries object.
	var apiList Entries
	err = json.NewDecoder(file).Decode(&apiList)
	if err != nil {
		panic("failed to decode JSON from file: " + err.Error())
	}
	file.Close() // do not defer since main func will remain open.

	// HTTP handler to receive incoming search requests.
	http.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {
		// Only allow GET requests
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Decode incoming search request off the query parameters map.
		searchReq := new(SearchRequest)
		err := schema.NewDecoder().Decode(searchReq, req.URL.Query())
		if err != nil {
			http.Error(w, "server failed to parse request: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Holds our matching entries that met the search parameters.
		var results []Entry

		// Loop through our APIs seeing if our search parameters match any in our list, appending them to the return
		// object if so.
		for _, e := range apiList.Entries {
			if checkEntryMatches(e, searchReq) {
				results = append(results, e)
			}
		}

		// Set content-type.
		w.Header().Set("Content-Type", "json/application")

		// Encode a new Entries object as our return response to the client for the search results. 200 is implied.
		err = json.NewEncoder(w).Encode(Entries{
			Count:   len(results),
			Entries: results,
		})
		if err != nil {
			http.Error(w, "server failed to encode response object: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Println("listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
