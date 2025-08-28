package internal

import "os"

func Ptr[T any](v T) *T {
	return &v
}

func FromEnvWithDefault(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
