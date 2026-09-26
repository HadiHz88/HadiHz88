package main

import (
	"bytes"
	"fmt"
	"strings"
)

// replaceBlock swaps everything between the NAME:START and NAME:END markers,
// leaving every other byte of doc untouched.
func replaceBlock(doc []byte, name, body string) ([]byte, error) {
	start := []byte("<!-- " + name + ":START -->")
	end := []byte("<!-- " + name + ":END -->")
	i := bytes.Index(doc, start)
	j := bytes.Index(doc, end)
	if i < 0 || j < 0 || j < i {
		return nil, fmt.Errorf("markers %s:START/%s:END not found", name, name)
	}
	i += len(start)

	var out bytes.Buffer
	out.Grow(len(doc) + len(body))
	out.Write(doc[:i])
	out.WriteString("\n")
	if body = strings.TrimSpace(body); body != "" {
		out.WriteString(body)
		out.WriteString("\n")
	}
	out.Write(doc[j:])
	return out.Bytes(), nil
}
