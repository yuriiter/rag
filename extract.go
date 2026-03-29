package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/taylorskalyo/goreader/epub"
)

func FindFiles(pattern string) []string {
	var files []string
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		rootDir := "."
		if parts[0] != "" {
			rootDir = parts[0]
		}
		suffix := strings.TrimPrefix(pattern, rootDir)
		suffix = strings.TrimPrefix(suffix, "**")
		suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

		filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				match, _ := filepath.Match(suffix, filepath.Base(path))
				if suffix == "" || match || strings.HasSuffix(filepath.Base(path), strings.TrimPrefix(suffix, "*")) {
					files = append(files, path)
				}
			}
			return nil
		})
	} else {
		f, _ := filepath.Glob(pattern)
		files = append(files, f...)
	}
	return files
}

func ExtractContent(source string) (text string, err error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return stripTags(string(b)), nil
	}

	ext := strings.ToLower(filepath.Ext(source))
	switch ext {
	case ".txt", ".md", ".go", ".js", ".json", ".py", ".html", ".css", ".java", ".c", ".h", ".cpp":
		b, err := os.ReadFile(source)
		return string(b), err
	case ".pdf":
		f, r, err := pdf.Open(source)
		if err != nil {
			return "", err
		}
		defer f.Close()
		var sb strings.Builder
		for i := 1; i <= r.NumPage(); i++ {
			p := r.Page(i)
			if !p.V.IsNull() {
				if t, err := p.GetPlainText(nil); err == nil {
					sb.WriteString(t + "\n")
				}
			}
		}
		return sb.String(), nil
	case ".docx":
		return parseDocx(source)
	case ".xlsx":
		return parseXlsx(source)
	case ".epub":
		rc, err := epub.OpenReader(source)
		if err != nil {
			return "", err
		}
		defer rc.Close()
		var sb strings.Builder
		if len(rc.Rootfiles) > 0 {
			for _, item := range rc.Rootfiles[0].Manifest.Items {
				if strings.Contains(item.MediaType, "html") {
					f, err := item.Open()
					if err != nil {
						continue
					}
					b, _ := io.ReadAll(f)
					f.Close()
					sb.WriteString(stripTags(string(b)) + "\n")
				}
			}
		}
		return sb.String(), nil
	}
	return "", fmt.Errorf("unsupported file type: %s", ext)
}

func parseDocx(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	var sb strings.Builder
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			dec := xml.NewDecoder(rc)
			for {
				t, _ := dec.Token()
				if t == nil {
					break
				}
				if se, ok := t.(xml.StartElement); ok && se.Name.Local == "t" {
					var s string
					dec.DecodeElement(&s, &se)
					sb.WriteString(s + " ")
				}
				if se, ok := t.(xml.StartElement); ok && (se.Name.Local == "p" || se.Name.Local == "br") {
					sb.WriteString("\n")
				}
			}
		}
	}
	return sb.String(), nil
}

func parseXlsx(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	var sb strings.Builder
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			dec := xml.NewDecoder(rc)
			for {
				t, _ := dec.Token()
				if t == nil {
					break
				}
				if se, ok := t.(xml.StartElement); ok && se.Name.Local == "t" {
					var s string
					dec.DecodeElement(&s, &se)
					sb.WriteString(s + "\n")
				}
			}
		}
	}
	return sb.String(), nil
}

func stripTags(c string) string {
	var sb strings.Builder
	in := false
	for _, r := range c {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
