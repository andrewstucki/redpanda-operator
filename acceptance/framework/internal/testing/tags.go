package testing

import (
	"context"
)

// CleanupFunc is a function run during a cleanup execution.
type CleanupFunc func(context.Context) error

// TagHandler is a function that processes a tag and returns a cleanup function.
type TagHandler func(context.Context) (func(context.Context) error, error)

type TagRegistry struct {
	tags map[string]TagHandler
}

func NewTagRegistry() *TagRegistry {
	return &TagRegistry{
		tags: make(map[string]TagHandler),
	}
}

func (r *TagRegistry) Register(tag string, handler TagHandler) {
	r.tags[tag] = handler
}

func (r *TagRegistry) Handlers(tags []string) []TagHandler {
	handlers := []TagHandler{}

	for _, tag := range tags {
		if handler, ok := r.tags[tag]; ok {
			handlers = append(handlers, handler)
		}
	}

	return handlers
}
