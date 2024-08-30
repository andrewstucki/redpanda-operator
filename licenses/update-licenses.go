// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

// go:build updatelicenses
// build updatelicenses
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	_ "embed"
)

type delimiter struct {
	top    string
	mid    string
	bottom string
	match  func(path string) bool
}

type templateData struct {
	Year int
}

var (
	//go:embed BSL.header.tmpl
	licenseHeader string

	//go:embed BSL.md.tmpl
	bslLicense string

	inlineSlashDelimiter = delimiter{mid: "//"}
	inlineHashDelimiter  = delimiter{mid: "#"}
	helmDelimiter        = delimiter{top: "{{/*", bottom: "*/}}", match: func(path string) bool {
		return strings.Contains(path, "charts") && strings.Contains(path, "templates") && !strings.Contains(path, "test")
	}}
	boilerplateGoHeaderDelimiter = delimiter{mid: "//", match: func(path string) bool {
		return path == "licenses/boilerplate.go.txt"
	}}

	licenseTemplate     *template.Template
	bslLicenseTemplate  *template.Template
	licenseTemplateData = &templateData{
		Year: time.Now().Year(),
	}
	extensions = map[string][]delimiter{
		".go":   []delimiter{inlineSlashDelimiter},
		".txt":  []delimiter{boilerplateGoHeaderDelimiter},
		".yml":  []delimiter{helmDelimiter},
		".yaml": []delimiter{helmDelimiter},
	}
	ignore = []func(path string) bool{
		func(path string) bool { return strings.Contains(path, "zz_generated") },
	}
)

func init() {
	licenseTemplate = template.Must(template.New("").Parse(licenseHeader))
	bslLicenseTemplate = template.Must(template.New("").Parse(bslLicense))
}

func main() {
	ch := make(chan *file, 1000)
	done := make(chan struct{})

	var groupErr error

	go func() {
		defer close(done)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var errOnce sync.Once
		var wg sync.WaitGroup

		for f := range ch {
			f := f
			wg.Add(1)
			go func() {
				defer wg.Done()

				err := f.process(ctx)
				if err != nil {
					errOnce.Do(func() {
						groupErr = err
						cancel()
					})
				}
			}()
		}
		wg.Wait()
	}()

	if err := walk(ch); err != nil {
		log.Printf("error processing files: %v", groupErr)
		os.Exit(1)
	}

	if groupErr != nil {
		log.Printf("error processing files: %v", groupErr)
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := bslLicenseTemplate.Execute(&buf, licenseTemplateData); err != nil {
		log.Printf("error processing files: %v", err)
		os.Exit(1)
	}

	if err := os.WriteFile("licenses/BSL.md", buf.Bytes(), 0644); err != nil {
		log.Printf("error processing files: %v", err)
		os.Exit(1)
	}

	log.Printf("finished processing files")
}

type file struct {
	extension string
	path      string
	mode      os.FileMode
}

func (f *file) process(ctx context.Context) error {
	log.Printf("Processing %q", f.path)

	data, err := os.ReadFile(f.path)
	if err != nil {
		return err
	}

	var delimiterToUse delimiter
	var found bool
	for _, d := range extensions[f.extension] {
		if d.match == nil || d.match(f.path) {
			delimiterToUse = d
			found = true
			break
		}
	}

	if !found {
		// skip
		return nil
	}

	var buf bytes.Buffer
	if err := licenseTemplate.Execute(&buf, licenseTemplateData); err != nil {
		return err
	}

	var out bytes.Buffer
	if delimiterToUse.top != "" {
		if _, err := fmt.Fprintln(&out, delimiterToUse.top); err != nil {
			return err
		}
	}
	s := bufio.NewScanner(&buf)
	for s.Scan() {
		mid := delimiterToUse.mid
		if mid != "" {
			mid += " "
		}

		if _, err := fmt.Fprintln(&out, strings.TrimRightFunc(mid+s.Text(), unicode.IsSpace)); err != nil {
			return err
		}
	}
	if delimiterToUse.bottom != "" {
		if _, err := fmt.Fprintln(&out, delimiterToUse.bottom); err != nil {
			return err
		}
	}

	s = bufio.NewScanner(bytes.NewReader(data))
	inLicense := false
	lines := 0
	for s.Scan() {
		lines++

		text := s.Text()

		if lines == 1 {
			if delimiterToUse.top != "" {
				if s.Text() == delimiterToUse.top {
					inLicense = true
					continue
				}
			} else if strings.HasPrefix(strings.ReplaceAll(text, " ", ""), delimiterToUse.mid+"Copyright") {
				inLicense = true
				continue
			}
		} else {
			if inLicense {
				if delimiterToUse.bottom != "" {
					if s.Text() == delimiterToUse.bottom {
						inLicense = false
						continue
					}
				} else if !strings.HasPrefix(text, delimiterToUse.mid) {
					inLicense = false
				}
			}
		}

		if inLicense {
			continue
		}

		if _, err := fmt.Fprintln(&out, text); err != nil {
			return err
		}
	}

	return os.WriteFile(f.path, out.Bytes(), f.mode)
}

func walk(ch chan<- *file) error {
	return filepath.Walk(".", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%s error: %v", path, err)
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		extension := filepath.Ext(path)

		if !fileIgnored(path) && fileMatches(extension) {
			ch <- &file{extension, path, fi.Mode()}
			return nil
		}
		return nil
	})
}

func fileIgnored(path string) bool {
	for _, shouldIgnore := range ignore {
		if shouldIgnore(path) {
			return true
		}
	}

	return false
}

func fileMatches(extension string) bool {
	for ext := range extensions {
		if extension == ext {
			return true
		}
	}
	return false
}
