package testing

import (
	"context"
	"sort"
	"strings"
)

// CleanupFunc is a function run during a cleanup execution.
type CleanupFunc func(context.Context) error

// TagHandler is a function that processes a tag and returns a cleanup function.
type TagHandler func(context.Context, string) (func(context.Context) error, error)

type ParsedTagHandler struct {
	Priority int
	Suffix   string
	Handler  TagHandler
}

type PriorityTagHandler struct {
	Priority int
	Handler  TagHandler
}

type TagRegistry struct {
	tags map[string]PriorityTagHandler
}

func NewTagRegistry() *TagRegistry {
	return &TagRegistry{
		tags: make(map[string]PriorityTagHandler),
	}
}

func (r *TagRegistry) Register(tag string, priority int, handler TagHandler) {
	r.tags[tag] = PriorityTagHandler{
		Handler:  handler,
		Priority: priority,
	}
}

func (r *TagRegistry) Handlers(tags []string) []ParsedTagHandler {
	handlers := []ParsedTagHandler{}

	for _, tag := range tags {
		tokens := strings.SplitN(tag, ":", 2)
		tag = tokens[0]

		suffix := ""
		if len(tokens) == 2 {
			suffix = tokens[1]
		}

		if handler, ok := r.tags[tag]; ok {
			handlers = append(handlers, ParsedTagHandler{
				Suffix:   suffix,
				Priority: handler.Priority,
				Handler:  handler.Handler,
			})
		}
	}

	sort.SliceStable(handlers, func(i, j int) bool {
		a, b := handlers[i], handlers[j]
		return a.Priority < b.Priority
	})

	return handlers
}
