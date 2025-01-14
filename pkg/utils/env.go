package utils

import "os"

func GetEnv(name, fallback string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		return fallback
	}
	return value
}
