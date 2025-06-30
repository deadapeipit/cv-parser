package main

import (
	"bytes"
	"cv-parser/model"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
)

func containsAny(line string, keywords []string) bool {
	for _, k := range keywords {
		if strings.Contains(line, k) {
			return true
		}
	}
	return false
}

func isLikelyBullet(line string) bool {
	return strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•")
}

func extractYears(line string) (string, string) {
	if match := yearRangeRegex.FindString(line); match != "" {
		years := strings.FieldsFunc(match, func(r rune) bool { return r == '-' || r == '–' })
		if len(years) == 2 {
			return strings.TrimSpace(years[0]), strings.TrimSpace(years[1])
		}
	}
	return "", ""
}

func isSectionHeader(line string) bool {
	l := strings.ToLower(strings.TrimSpace(line))
	return strings.Contains(l, "education") || strings.Contains(l, "experience") || strings.Contains(l, "skills") || strings.Contains(l, "projects")
}

func extractEmail(text string) string {
	return emailRegex.FindString(text)
}

func extractPhone(text string) string {
	return phoneRegex.FindString(text)
}

func extractSkills(text string) []string {
	var found []string
	for _, skill := range knownSkills {
		if strings.Contains(strings.ToLower(text), strings.ToLower(skill)) {
			found = append(found, skill)
		}
	}
	return found
}

func extractEducation(lines []string) []model.Education {
	var result []model.Education
	var block []string
	inSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.ToLower(line) == "education" {
			inSection = true
			continue
		}
		if inSection {
			if isSectionHeader(line) && len(block) > 0 {
				entry := parseEducationBlock(block)
				if entry.Degree != "" || entry.Institution != "" {
					result = append(result, entry)
				}
				block = nil
				break
			}
			if line == "" {
				if len(block) > 0 {
					entry := parseEducationBlock(block)
					if entry.Degree != "" || entry.Institution != "" {
						result = append(result, entry)
					}
					block = nil
				}
				continue
			}
			block = append(block, line)
		}
	}

	if len(block) > 0 {
		entry := parseEducationBlock(block)
		if entry.Degree != "" || entry.Institution != "" {
			result = append(result, entry)
		}
	}
	return result
}

func extractLinks(text string) (linkedin string, github string, others map[string]string) {
	others = make(map[string]string)

	for _, match := range urlRegex.FindAllString(text, -1) {
		switch {
		case linkedinRegex.MatchString(match):
			if linkedin == "" {
				linkedin = match
			}
		case githubRegex.MatchString(match):
			if github == "" {
				github = match
			}
		default:
			key := getDomain(match)
			others[key] = match
		}
	}
	return
}

func getDomain(url string) string {
	url = strings.Replace(url, "https://", "", 1)
	url = strings.Replace(url, "http://", "", 1)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

func parseEducationBlock(lines []string) model.Education {
	entry := model.Education{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		if entry.Start == "" && entry.End == "" {
			start, end := extractYears(line)
			if start != "" && end != "" {
				entry.Start = start
				entry.End = end
				continue
			}
		}
		if entry.Degree == "" && containsAny(lineLower, educationKeywords) {
			entry.Degree = line
			continue
		}
		if entry.Institution == "" && containsAny(lineLower, institutionKeywords) {
			entry.Institution = line
			continue
		}
	}
	return entry
}

func extractExperience(lines []string) []model.Experience {
	var result []model.Experience
	var block []string
	inSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.ToLower(line) == "experience" {
			inSection = true
			continue
		}
		if inSection {
			if isSectionHeader(line) && len(block) > 0 {
				result = append(result, parseExperienceBlock(block))
				block = nil
				break
			}
			if line == "" {
				if len(block) > 0 {
					result = append(result, parseExperienceBlock(block))
					block = nil
				}
				continue
			}
			block = append(block, line)
		}
	}
	if len(block) > 0 {
		result = append(result, parseExperienceBlock(block))
	}
	return result
}

func parseExperienceBlock(lines []string) model.Experience {
	entry := model.Experience{}
	detailMode := false

	for i, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Date first
		if entry.Start == "" && entry.End == "" {
			start, end := extractYears(line)
			if start != "" && end != "" {
				entry.Start = start
				entry.End = end
				continue
			}
		}

		// Position (usually before company)
		if entry.Position == "" && containsAny(lineLower, positionKeywords) {
			entry.Position = line
			continue
		}

		// Company name
		if entry.Company == "" && containsAny(lineLower, companyKeywords) {
			entry.Company = line
			continue
		}

		// Detail bullets or paragraphs
		if isLikelyBullet(line) || (entry.Company != "" && entry.Position != "" && i > 2) {
			entry.Details = append(entry.Details, strings.TrimPrefix(line, "- "))
			detailMode = true
		} else if detailMode {
			entry.Details = append(entry.Details, line) // catch wrapped lines
		}
	}
	return entry
}

func parseResume(text string) model.Resume {
	lines := strings.Split(text, "\n")
	linkedin, github, others := extractLinks(text)
	return model.Resume{
		Email:      extractEmail(text),
		Phone:      extractPhone(text),
		Skills:     extractSkills(text),
		LinkedIn:   linkedin,
		GitHub:     github,
		OtherLinks: others,
		Education:  extractEducation(lines),
		Experience: extractExperience(lines),
	}
}

func extractTextFromPDF(file multipart.File) (string, error) {
	temp, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(temp.Name())
	io.Copy(temp, file)
	temp.Close()
	f, r, err := pdf.Open(temp.Name())
	if err != nil {
		return "", err
	}
	defer f.Close()
	var buf bytes.Buffer
	numPages := r.NumPage()
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err == nil {
			buf.WriteString(content)
		}
	}
	return buf.String(), nil
}

func extractTextFromDocx(file multipart.File) (string, error) {
	temp, err := os.CreateTemp("", "*.docx")
	if err != nil {
		return "", err
	}
	defer os.Remove(temp.Name())
	io.Copy(temp, file)
	temp.Close()
	r, err := docx.ReadDocxFile(temp.Name())
	if err != nil {
		return "", err
	}
	docx1 := r.Editable()
	return docx1.GetContent(), nil
}
