package profile

import (
	"reflect"
	"testing"
	"time"
)

func TestParse(t *testing.T) {

	tests := []struct {
		name    string
		md      string
		want    *Note
		wantErr bool
	}{
		{
			name: "ValidWithFrontMatter",
			md:   validWithFrontMatter,
			want: &Note{
				Name:      "foo",
				Tags:      []string{"test", "foo"},
				CreatedAt: sampleCreatedAt,
				UpdatedAt: sampleUpdatedAt,
				Content:   "# Foo\n\nSome notes",
			},
		},
		{
			name:    "InvalidFrontMatter",
			md:      "---\n:\n---",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.md))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	sampleCreatedAt = time.Unix(1644811695, 0)
	sampleUpdatedAt = time.Unix(1644812695, 0)
)

const (
	validWithFrontMatter = `---
id: 3
name: foo
tags:
- test
- foo
created_at: 2022-02-14T09:38:15+05:30
updated_at: 2022-02-14T09:54:55+05:30
---

# Foo

Some notes
`
)
