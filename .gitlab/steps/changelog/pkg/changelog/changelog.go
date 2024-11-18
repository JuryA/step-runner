package changelog

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const expectedFormat = "## vMAJOR.MINOR.PATCH (YYYY-MM-DD)" // example: ## v17.5.0 (2024-10-17)
var expectedFormatRe = regexp.MustCompile(`## v(.+)\.(.+)\.(.+) \((\d{4}-\d{2}-\d{2})\)`)

type Changelog struct {
	contents []byte
}

func New(contents []byte) *Changelog {
	return &Changelog{contents: contents}
}

func (c *Changelog) LatestVersion() (*Version, error) {
	versions, err := c.read()

	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("failed to find any versions in changelog")
	}

	return versions[0], nil
}

func (c *Changelog) read() ([]*Version, error) {
	scanner := bufio.NewScanner(bytes.NewReader(c.contents))
	var versions []*Version
	var changes []string
	var current []string

	for scanner.Scan() {
		line := scanner.Text()

		parts := expectedFormatRe.FindStringSubmatch(scanner.Text())

		switch {
		case len(parts) != 5 && current == nil:
			return nil, fmt.Errorf("must start with version, line '%s' does not conform to expected format '%s'", line, expectedFormat)
		case len(parts) != 5 && strings.HasPrefix(line, "## "):
			return nil, fmt.Errorf("header line '%s' does not conform to expected format '%s'", line, expectedFormat)
		case len(parts) != 5:
			changes = append(changes, line)
		case len(parts) == 5:
			if current != nil {
				versions = append(versions, NewVersion(current[1], current[2], current[3], changes))
			}

			current = parts
			changes = make([]string, 0)
		}

		err := scanner.Err()

		if err != nil {
			return nil, fmt.Errorf("failed to read changelog: %w", err)
		}
	}

	if current != nil {
		versions = append(versions, NewVersion(current[1], current[2], current[3], changes))
	}

	return versions, nil
}
