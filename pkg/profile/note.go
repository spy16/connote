package profile

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
	Name      string    `json:"name" yaml:"name"`
	Tags      []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Content   string    `json:"content,omitempty" yaml:"content,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

func (nt *Note) Validate() error {
	nt.Name = strings.TrimSpace(nt.Name)
	if nt.CreatedAt.IsZero() {
		nt.CreatedAt = time.Now()
		nt.UpdatedAt = nt.CreatedAt
	}

	tagSet := map[string]string{}
	for _, tag := range nt.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			k, v := splitTag(tag)
			tagSet[k] = v
		}
	}

	nt.Tags = nil
	for k, v := range tagSet {
		if v == "" {
			nt.Tags = append(nt.Tags, k)
		} else {
			nt.Tags = append(nt.Tags, fmt.Sprintf("%s:%s", k, v))
		}
	}

	if !nameExp.MatchString(nt.Name) {
		return fmt.Errorf("invalid name: '%s'", nt.Name)
	}
	return nil
}

// ToMarkdown returns article in mark-down format. name, attribs, etc.
// are added as front-matter.
func (nt *Note) ToMarkdown() []byte {
	content := nt.Content
	nt.Content = ""

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if err := yaml.NewEncoder(&buf).Encode(nt); err != nil {
		panic(fmt.Errorf("failed to encode yaml: %v", err))
	}
	buf.WriteString("---\n\n")
	buf.WriteString(content)

	return buf.Bytes()
}

func (nt *Note) FromMD(d []byte) error {
	n, err := Parse(d)
	if err != nil {
		return err
	}
	*nt = *n
	return nil
}
