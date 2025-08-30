package ginkgoextend

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"
)

// MatchNonZero recursively compares actual to expected, only checking non-zero fields in expected.
func MatchNonZero(expected interface{}) types.GomegaMatcher {
	return &nonZeroMatcher{expected: expected}
}

type nonZeroMatcher struct {
	expected interface{}
}

func (m *nonZeroMatcher) Match(actual interface{}) (bool, error) {
	return matchNonZeroRecursive(reflect.ValueOf(m.expected), reflect.ValueOf(actual))
}

func matchNonZeroRecursive(expVal, actVal reflect.Value) (bool, error) {
	if !expVal.IsValid() {
		return true, nil // zero expected → ignore
	}

	// Dereference pointers
	for expVal.Kind() == reflect.Ptr {
		if expVal.IsNil() {
			return true, nil // zero pointer → ignore
		}
		expVal = expVal.Elem()
	}
	for actVal.Kind() == reflect.Ptr {
		if actVal.IsNil() {
			actVal = reflect.Zero(expVal.Type())
		} else {
			actVal = actVal.Elem()
		}
	}

	if !actVal.IsValid() {
		return false, errors.New("actual value not valid")
	}

	if expVal.Kind() != actVal.Kind() {
		return false, errors.New("actual value and expected value kind mismatch")
	}

	switch expVal.Kind() {
	case reflect.Struct:
		for i := 0; i < expVal.NumField(); i++ {
			fieldExp := expVal.Field(i)
			fieldAct := actVal.Field(i)
			// always check booleans (even though they also have a default value)
			// TODO make configurable
			if !fieldExp.IsZero() /* || fieldExp.Kind() == reflect.Bool */ {
				ok, err := matchNonZeroRecursive(fieldExp, fieldAct)
				if err != nil || !ok {
					return false, err
				}
			}
		}
		return true, nil
	case reflect.Slice, reflect.Array:
		if expVal.Len() == 0 {
			return true, nil
		}
		if expVal.Len() != actVal.Len() {
			return false, errors.New("slice length mismatch")
		}
		for i := 0; i < expVal.Len(); i++ {
			ok, err := matchNonZeroRecursive(expVal.Index(i), actVal.Index(i))
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	case reflect.Map:
		if expVal.Len() == 0 {
			return true, nil
		}
		for _, key := range expVal.MapKeys() {
			valExp := expVal.MapIndex(key)
			valAct := actVal.MapIndex(key)
			if !valExp.IsZero() {
				if !valAct.IsValid() {
					return false, errors.New("map actual value not zero")
				}
				ok, err := matchNonZeroRecursive(valExp, valAct)
				if err != nil || !ok {
					return false, err
				}
			}
		}
		return true, nil
	default:
		// Scalar comparison
		// Only handle comparable scalars
		if !expVal.Type().Comparable() || !actVal.Type().Comparable() {
			return false, fmt.Errorf("cannot compare values of types %s and %s", expVal.Type(), actVal.Type())
		}
		if expVal.Interface() != actVal.Interface() {
			return false, fmt.Errorf("scalar value comparison failed")
		}
		return true, nil
	}
}

func (m *nonZeroMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nto match non-zero fields of\n\t%#v", actual, m.expected)
}

func (m *nonZeroMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match non-zero fields of\n\t%#v", actual, m.expected)
}
