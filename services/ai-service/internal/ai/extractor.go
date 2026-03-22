package ai

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractTextFromFile takes a byte slice and a filename, identifies the file type,
// and extracts all possible text from it to be fed into an LLM.
func ExtractTextFromFile(fileBytes []byte, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return extractTextFromPDF(fileBytes)
	case ".docx":
		return extractTextFromDocx(fileBytes)
	case ".pptx":
		return extractTextFromPptx(fileBytes)
	case ".txt", ".md", ".csv":
		return string(fileBytes), nil
	default:
		return string(fileBytes), nil // Try as plain text
	}
}

// extractTextFromPDF uses ledongthuc/pdf to parse simple PDF text
func extractTextFromPDF(fileBytes []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // Skip problematic pages
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// extractTextFromDocx extracts XML content from docx (since it's a zip file)
func extractTextFromDocx(fileBytes []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", err
			}
			
			// Extremely naive XML to Text fallback
			// Since we just need the text for LLM, strip the XML tags.
			text := string(content)
			text = stripXMLTags(text)
			textBuilder.WriteString(text)
			break
		}
	}
	
	if textBuilder.Len() == 0 {
		return "", fmt.Errorf("could not find document.xml in docx")
	}
	
	return textBuilder.String(), nil
}

// Helper to strip XML tags quickly
func stripXMLTags(content string) string {
	var builder strings.Builder
	inTag := false
	for _, char := range content {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			builder.WriteString(" ") // Add space to prevent word joining
			continue
		}
		if !inTag {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

// extractTextFromPptx extracts text from pptx files by reading ppt/slides/slide*.xml
func extractTextFromPptx(fileBytes []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	
	// Collect all slides
	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			
			// Basic XML to Text fallback
			text := string(content)
			text = stripXMLTags(text)
			textBuilder.WriteString(text)
			textBuilder.WriteString("\n\n")
		}
	}
	
	if textBuilder.Len() == 0 {
		return "", fmt.Errorf("could not find any presentation slides in pptx")
	}
	
	return textBuilder.String(), nil
}
