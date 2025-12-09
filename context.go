package axiom

import (
	"context"
	"fmt"
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

func WithContextRaw(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.Raw = ctx
	}
}

func WithContextHTTP(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.HTTP = ctx
	}
}

func WithContextGRPC(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.GRPC = ctx
	}
}

func WithContextKafka(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.Kafka = ctx
	}
}

func WithContextData(key string, value any) ContextOption {
	return func(c *Context) {
		if c.Data == nil {
			c.Data = map[string]any{}
		}

		c.Data[key] = value
	}
}

func GetContextValue[T any](c *Context, key string) (T, bool) {
	v, ok := c.Data[key]
	if !ok {
		var zero T
		return zero, false
	}

	out, ok := v.(T)
	return out, ok
}

func MustContextValue[T any](c *Context, key string) T {
	v, ok := GetContextValue[T](c, key)
	if !ok {
		panic(fmt.Sprintf("context: expected value for key %q of type %T", key, *new(T)))
	}
	return v
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
