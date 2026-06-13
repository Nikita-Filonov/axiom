package axiom

import (
	"reflect"
	"strings"
	"testing"
)

type Suite struct {
	RootT  *testing.T
	SubT   *testing.T
	Runner *Runner
}

type SuiteOption func(*Suite)

func WithSuiteRunner(runner *Runner) SuiteOption {
	return func(s *Suite) { s.Runner = runner }
}

func findEmbeddedSuite(suiteValue reflect.Value) *Suite {
	suiteStruct := suiteValue.Elem()
	suiteStructType := suiteStruct.Type()

	suiteType := reflect.TypeOf(Suite{})
	suitePointerType := reflect.TypeOf((*Suite)(nil))

	for i := 0; i < suiteStruct.NumField(); i++ {
		structField := suiteStructType.Field(i)
		if !structField.Anonymous {
			continue
		}

		field := suiteStruct.Field(i)

		switch field.Type() {
		case suiteType:
			if field.CanAddr() {
				return field.Addr().Interface().(*Suite)
			}

		case suitePointerType:
			if field.IsNil() {
				field.Set(reflect.New(suiteType))
			}

			return field.Interface().(*Suite)
		}
	}

	return nil
}

func discoverSuiteTests(suiteValue reflect.Value) []string {
	suiteType := suiteValue.Type()
	tests := make([]string, 0)

	for i := 0; i < suiteType.NumMethod(); i++ {
		method := suiteType.Method(i)

		if !strings.HasPrefix(method.Name, "Test") {
			continue
		}

		if method.Type.NumIn() != 1 || method.Type.NumOut() != 0 {
			continue
		}

		tests = append(tests, method.Name)
	}

	return tests
}

func RunSuite(t *testing.T, suite any, options ...SuiteOption) {
	if t == nil {
		panic("suite: nil *testing.T")
	}

	suiteValue := reflect.ValueOf(suite)
	if !suiteValue.IsValid() {
		panic("suite: suite must be a non-nil pointer to a struct")
	}
	if suiteValue.Kind() != reflect.Ptr {
		panic("suite: suite must be a non-nil pointer to a struct")
	}
	if suiteValue.IsNil() {
		panic("suite: suite must be a non-nil pointer to a struct")
	}
	if suiteValue.Elem().Kind() != reflect.Struct {
		panic("suite: suite must be a pointer to a struct")
	}

	baseSuite := findEmbeddedSuite(suiteValue)
	if baseSuite == nil {
		panic("suite: suite must embed axiom.Suite")
	}

	baseSuite.RootT = t
	for _, option := range options {
		option(baseSuite)
	}
	if baseSuite.Runner == nil {
		baseSuite.Runner = NewRunner()
	}

	baseSuite.Runner.ApplyStart()
	t.Cleanup(baseSuite.Runner.ApplyFinish)

	for _, test := range discoverSuiteTests(suiteValue) {
		t.Run(test, func(st *testing.T) {
			baseSuite.SubT = st
			defer func() { baseSuite.SubT = nil }()

			suiteValue.MethodByName(test).Call(nil)
		})
	}
}

func (s *Suite) RunCase(c Case, a TestAction) {
	if s == nil {
		panic("suite: nil Suite")
	}
	if s.Runner == nil {
		panic("suite: runner is not configured")
	}
	if s.SubT == nil {
		panic("suite: nil *testing.T")
	}

	s.Runner.runCase(s.SubT, c, a)
}
