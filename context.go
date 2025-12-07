package axiom

import (
	"context"
)

type Context struct {
	Raw   context.Context
	GRPC  context.Context
	HTTP  context.Context
	Kafka context.Context

	Data map[string]any
}

type ContextOption func(*Context)

func NewContext(options ...ContextOption) Context {
	c := Context{}
	for _, option := range options {
		option(&c)
	}

	return c
}

func WithContextData(key string, value any) ContextOption {
	return func(c *Context) {
		if c.Data == nil {
			c.Data = map[string]any{}
		}

		c.Data[key] = value
	}
}

func (c *Context) Join(other Context) Context {
	result := Context{
		Raw:   c.Raw,
		GRPC:  c.GRPC,
		HTTP:  c.HTTP,
		Kafka: c.Kafka,
		Data:  map[string]any{},
	}
	for k, v := range c.Data {
		result.Data[k] = v
	}

	if other.Raw != nil {
		result.Raw = other.Raw
	}
	if other.GRPC != nil {
		result.GRPC = other.GRPC
	}
	if other.HTTP != nil {
		result.HTTP = other.HTTP
	}
	if other.Kafka != nil {
		result.Kafka = other.Kafka
	}

	for k, v := range other.Data {
		result.Data[k] = v
	}

	return result
}

func (c *Context) Normalize() {
	if c.Raw == nil {
		c.Raw = context.Background()
	}
	if c.GRPC == nil {
		c.GRPC = c.Raw
	}
	if c.HTTP == nil {
		c.HTTP = c.Raw
	}
	if c.Kafka == nil {
		c.Kafka = c.Raw
	}
	if c.Data == nil {
		c.Data = map[string]any{}
	}
}
