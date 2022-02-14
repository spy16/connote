package fs

import (
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/spy16/connote/note"
)

const excludeTagPrefix = "~"

type noteIndex struct {
	Notes map[string]indexNode `json:"notes"`
	dirty bool
}

type indexNode struct {
	ID        int                 `json:"id"`
	Name      string              `json:"name"`
	Tags      map[string]struct{} `json:"tags"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

func (n indexNode) isMatch(q note.Query) bool {
	if q.NameRegex != "" {
		re, err := regexp.Compile(q.NameRegex)
		if err != nil {
			logrus.Warnf("ignoring invalid name pattern '%s': %v", q.NameRegex, err)
		} else if !re.MatchString(n.Name) {
			return false
		}
	}

	for _, tag := range q.Tags {
		if strings.HasPrefix(tag, excludeTagPrefix) {
			tag = strings.TrimPrefix(tag, excludeTagPrefix)
			if _, hasTag := n.Tags[tag]; hasTag {
				return false
			}
		} else {
			if _, hasTag := n.Tags[tag]; !hasTag {
				return false
			}
		}
	}

	if !q.CreatedBetween[0].IsZero() {
		if n.CreatedAt.Before(q.CreatedBetween[0]) {
			return false
		}
	}
	if !q.CreatedBetween[1].IsZero() {
		if n.CreatedAt.After(q.CreatedBetween[1]) {
			return false
		}
	}

	return true
}

func arrToSet(arr []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range arr {
		m[s] = struct{}{}
	}
	return m
}
