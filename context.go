package axiom

import (
	"context"
	"fmt"
)

type Context struct {
	Raw context.Context

	DB  context.Context
	MQ  context.Context
	RPC context.Context

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
	return func(c *Context) { c.Raw = ctx }
}

func WithContextDB(ctx context.Context) ContextOption {
	return func(c *Context) { c.DB = ctx }
}

func WithContextMQ(ctx context.Context) ContextOption {
	return func(c *Context) { c.MQ = ctx }
}

func WithContextRPC(ctx context.Context) ContextOption {
	return func(c *Context) { c.RPC = ctx }
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

func (c *Context) SetData(key string, value any) {
	if c.Data == nil {
		c.Data = map[string]any{}
	}
	c.Data[key] = value
}

func (c *Context) Join(other Context) Context {
	result := Context{
		Raw:  c.Raw,
		DB:   c.DB,
		MQ:   c.MQ,
		RPC:  c.RPC,
		Data: map[string]any{},
	}
	for k, v := range c.Data {
		result.Data[k] = v
	}

	if other.Raw != nil {
		result.Raw = other.Raw
	}
	if other.DB != nil {
		result.DB = other.DB
	}
	if other.MQ != nil {
		result.MQ = other.MQ
	}
	if other.RPC != nil {
		result.RPC = other.RPC
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
	if c.DB == nil {
		c.DB = c.Raw
	}
	if c.MQ == nil {
		c.MQ = c.Raw
	}
	if c.RPC == nil {
		c.RPC = c.Raw
	}
	if c.Data == nil {
		c.Data = map[string]any{}
	}
}
