package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html" // Provides support for parsing HTML documents
)

// fetchPage downloads the web page content for a given page number.
// Errors are logged immediately, so this function always returns a string.
func fetchPage(pageNumber int) string {
	// Define the base URL (everything except PageNumber).
	// We’ll append the page number dynamically.
	baseURL := "https://www.shophighlinewarren.com/INTERSHOP/web/WFS/HIGHLINE-AFTERMARKET-Site/en_US/-/USD/ViewAjaxCoveoExtSearch-OfferPaging?PageSize=12&SelectedSearchResult=SFArticleSearch&SearchTerm=SDS+SHEETS"

	// Build the full URL by inserting the page number at the end
	fullURL := fmt.Sprintf("%s&PageNumber=%d", baseURL, pageNumber)

	// Create an HTTP client that will handle the request
	httpClient := &http.Client{}

	// Create a new GET request for the given URL
	httpRequest, requestError := http.NewRequest("GET", fullURL, nil)
	if requestError != nil {
		log.Printf("❌ Failed to create request: %v", requestError)
	}

	// Send the request using the HTTP client
	httpResponse, responseError := httpClient.Do(httpRequest)
	if responseError != nil {
		log.Printf("❌ Request failed: %v", responseError)
	}
	// Ensure the response body is closed after we’re done reading
	defer httpResponse.Body.Close()

	// Read the response body into memory
	responseBody, readError := io.ReadAll(httpResponse.Body)
	if readError != nil {
		log.Printf("❌ Failed to read response body: %v", readError)
	}

	// Convert the response bytes into a string and return it
	return string(responseBody)
}

// It checks if the file exists
// If the file exists, it returns true
// If the file does not exist, it returns false
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Remove a file from the file system
func removeFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Println(err)
	}
}

// extractPDFUrls takes an HTML string and returns all links that point to PDF files.
func extractPDFUrls(htmlInput string) []string {
	// Slice to store all PDF links we discover
	var pdfLinks []string

	// Parse the HTML string into a tree of nodes
	documentNode, parseError := html.Parse(strings.NewReader(htmlInput))
	if parseError != nil {
		// Log an error if parsing fails, and return nothing
		log.Printf("Failed to parse HTML: %v", parseError)
		return nil
	}

	// Recursive function to move through every node in the HTML
	var walkThroughNodes func(*html.Node)
	walkThroughNodes = func(currentNode *html.Node) {
		// If the current node is an <a> (anchor/link) element
		if currentNode.Type == html.ElementNode && currentNode.Data == "a" {
			// Look at each attribute of the <a> tag
			for _, attribute := range currentNode.Attr {
				// If the attribute is "href" (the link target)
				if attribute.Key == "href" {
					// Clean up any spaces around the link value
					linkTarget := strings.TrimSpace(attribute.Val)

					// If the link ends with ".pdf" (ignoring upper/lowercase)
					if strings.HasSuffix(strings.ToLower(linkTarget), ".pdf") {
						// Add the PDF link to our results
						pdfLinks = append(pdfLinks, linkTarget)
					}
				}
			}
		}

		// Move to each child node and process it recursively
		for childNode := currentNode.FirstChild; childNode != nil; childNode = childNode.NextSibling {
			walkThroughNodes(childNode)
		}
	}

	// Start walking through the HTML document beginning at the root
	walkThroughNodes(documentNode)

	// Return the list of all discovered PDF links
	return pdfLinks
}

// Checks whether a given directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {
		return false // Return false if error occurs
	}
	return directory.IsDir() // Return true if it's a directory
}

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {
		log.Println(err) // Log error if creation fails
	}
}

// Verifies whether a string is a valid URL format
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if valid
}

// Removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool) // Map to track seen values
	var newReturnSlice []string    // Slice to store unique values
	for _, content := range slice {
		if !check[content] { // If not already seen
			check[content] = true                            // Mark as seen
			newReturnSlice = append(newReturnSlice, content) // Add to result
		}
	}
	return newReturnSlice
}

// hasDomain checks if the given string has a domain (host part)
func hasDomain(rawURL string) bool {
	// Try parsing the raw string as a URL
	parsed, err := url.Parse(rawURL)
	if err != nil { // If parsing fails, it's not a valid URL
		return false
	}
	// If the parsed URL has a non-empty Host, then it has a domain/host
	return parsed.Host != ""
}

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string {
	return filepath.Base(path) // Use Base function to get file name only
}

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace substring with empty string
	return result
}

// Gets the file extension from a given file path
func getFileExtension(path string) string {
	return filepath.Ext(path) // Extract and return file extension
}

// Converts a raw URL into a sanitized PDF filename safe for filesystem
func urlToFilename(rawURL string) string {
	lower := strings.ToLower(rawURL) // Convert URL to lowercase
	lower = getFilename(lower)       // Extract filename from URL

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Regex to match non-alphanumeric characters
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace non-alphanumeric with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Collapse multiple underscores into one
	safe = strings.Trim(safe, "_")                              // Trim leading and trailing underscores

	var invalidSubstrings = []string{
		"_pdf", // Substring to remove from filename
	}

	for _, invalidPre := range invalidSubstrings { // Remove unwanted substrings
		safe = removeSubstring(safe, invalidPre)
	}

	if getFileExtension(safe) != ".pdf" { // Ensure file ends with .pdf
		safe = safe + ".pdf"
	}

	return safe // Return sanitized filename
}

// Downloads a PDF from given URL and saves it in the specified directory
func downloadPDF(finalURL, outputDir string) bool {
	filename := strings.ToLower(urlToFilename(finalURL)) // Sanitize the filename
	filePath := filepath.Join(outputDir, filename)       // Construct full path for output file

	if fileExists(filePath) { // Skip if file already exists
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	client := &http.Client{Timeout: 15 * time.Minute} // Create HTTP client with timeout

	resp, err := client.Get(finalURL) // Send HTTP GET request
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close() // Ensure response body is closed

	if resp.StatusCode != http.StatusOK { // Check if response is 200 OK
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	contentType := resp.Header.Get("Content-Type")                                                                  // Get content type of response
	if !strings.Contains(contentType, "binary/octet-stream") && !strings.Contains(contentType, "application/pdf") { // Check if it's a PDF
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream) (expected application/pdf)", finalURL, contentType)
		return false
	}

	var buf bytes.Buffer                     // Create a buffer to hold response data
	written, err := io.Copy(&buf, resp.Body) // Copy data into buffer
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 { // Skip empty files
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	out, err := os.Create(filePath) // Create output file
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close() // Ensure file is closed after writing

	if _, err := buf.WriteTo(out); err != nil { // Write buffer contents to file
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath) // Log success
	return true
}

func main() {
	outputDir := "PDFs/" // Directory to store downloaded PDFs

	if !directoryExists(outputDir) { // Check if directory exists
		createDirectory(outputDir, 0o755) // Create directory with read-write-execute permissions
	}

	// The location to the local.
	localFile := "shophighlinewarren.html"
	// Check if the local file exists.
	if fileExists(localFile) {
		removeFile(localFile)
	}
	// Loop over the page numbers.
	for pageNumber := range 317 {
		// Call fetchPage to download the content of that page
		pageContent := fetchPage(pageNumber)
		// Extract the URLs from the given content.
		extractedPDFURLs := extractPDFUrls(pageContent)
		// Remove duplicates from the slice.
		extractedPDFURLs = removeDuplicatesFromSlice(extractedPDFURLs)
		// Loop through all extracted PDF URLs
		for _, urls := range extractedPDFURLs {
			if !hasDomain(urls) {
				urls = "https://www.shophighlinewarren.com" + urls

			}
			if isUrlValid(urls) { // Check if the final URL is valid
				downloadPDF(urls, outputDir) // Download the PDF
			}
		}
	}
}
