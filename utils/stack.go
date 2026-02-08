package utils

import (
	"runtime"
)

const (
	// LoggerPackageName is the package name of the logger package.
	// Used to skip internal frames when capturing caller information.
	LoggerPackageName = "github.com/edaniel30/loki-logger-go"
)

// GetCaller walks the call stack and returns the first caller outside of the logger package.
// This approach is more resilient to refactoring than hard-coded skip depths.
// Returns file path, line number, and ok status.
func GetCaller() (string, int, bool) {
	const maxDepth = 25 // Reasonable upper limit to prevent infinite loops

	// Track if we've seen any logger frames
	seenLoggerFrame := false

	for skip := 1; skip < maxDepth; skip++ {
		pc, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}

		// Get the function for this program counter
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		funcName := fn.Name()

		// Check if this is a logger package frame
		isLoggerFrame := containsPackage(funcName, LoggerPackageName)

		if isLoggerFrame {
			seenLoggerFrame = true
			continue
		}

		// If we've seen a logger frame and now we're outside, this is our caller
		if seenLoggerFrame {
			return file, line, true
		}
	}
	return "", 0, false
}

// containsPackage checks if a function name belongs to the given package.
func containsPackage(funcName, pkgName string) bool {
	// Function names have format: package/path.Type.Method or package/path.Function
	// We check if the function name starts with our package path
	return len(funcName) >= len(pkgName) && funcName[:len(pkgName)] == pkgName
}
