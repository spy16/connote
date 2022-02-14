package note

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	hrExp   = regexp.MustCompile("^-{3}-*$")
	idExp   = regexp.MustCompile("^[0-9]+$")
	nameExp = regexp.MustCompile("^[A-Za-z][A-Za-z0-9-_/:]+$")
)

// Parse parses the given markdown data and creates an instance
// of Note.
func Parse(md []byte) (*Note, error) {
	sc := bufio.NewScanner(bytes.NewReader(md))

	var frontMatter string
	var content string
	var readingFrontMatter bool
	for lineNo := 1; sc.Scan(); lineNo++ {
		line := sc.Text()
		if lineNo == 1 && hrExp.MatchString(line) {
			readingFrontMatter = true
		} else if readingFrontMatter {
			if hrExp.MatchString(line) {
				readingFrontMatter = false
			} else {
				frontMatter += "\n" + line
			}
		} else {
			content += "\n" + line
		}
	}

	var ar Note
	content = strings.TrimSpace(content)
	frontMatter = strings.TrimSpace(frontMatter)
	if err := yaml.Unmarshal([]byte(frontMatter), &ar); err != nil {
		return nil, err
	}
	ar.Content = content

	return &ar, nil
}

// Note represents a snippet of information with additional metadata.
type Note struct {
	ID        int       `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	Tags      []string  `json:"tags" yaml:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`

	Content string `json:"content" yaml:"content,omitempty"`
}

func (ar *Note) Validate() error {
	ar.Name = strings.TrimSpace(ar.Name)
	if ar.CreatedAt.IsZero() {
		ar.CreatedAt = time.Now()
		ar.UpdatedAt = ar.CreatedAt
	}

	tagSet := map[string]string{}
	for _, tag := range ar.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			k, v := splitTag(tag)
			tagSet[k] = v
		}
	}

	ar.Tags = nil
	for k, v := range tagSet {
		if v == "" {
			ar.Tags = append(ar.Tags, k)
		} else {
			ar.Tags = append(ar.Tags, fmt.Sprintf("%s:%s", k, v))
		}
	}

	if !nameExp.MatchString(ar.Name) {
		return fmt.Errorf("invalid name: '%s'", ar.Name)
	}
	return nil
}

// ToMarkdown returns article in mark-down format. name, attribs, etc.
// are added as front-matter.
func (ar *Note) ToMarkdown() []byte {
	content := ar.Content
	ar.Content = ""

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if err := yaml.NewEncoder(&buf).Encode(ar); err != nil {
		panic(fmt.Errorf("failed to encode yaml: %v", err))
	}
	buf.WriteString("---\n\n")
	buf.WriteString(content)

	return buf.Bytes()
}

func (ar Note) IsMatch(q Query) bool {
	return false
}

func splitTag(tag string) (k, v string) {
	pair := strings.SplitN(tag, ":", 2)
	key := pair[0]
	if len(pair) == 1 {
		return key, ""
	}
	return key, pair[1]
}
