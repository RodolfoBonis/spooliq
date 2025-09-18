package types

import (
	"runtime"
	"strconv"
	"strings"
)

// StackTrace represents a stack trace for error handling.
type StackTrace []uintptr

func (s *StackTrace) String() string {
	var sb strings.Builder
	stack := *s
	for k := range stack {
		v := stack[k] - 1
		f := runtime.FuncForPC(v)
		file, line := f.FileLine(v)

		sb.WriteString(f.Name())
		sb.WriteString("\n\t")
		sb.WriteString(file)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(line))
		sb.WriteString("\n")
	}

	result := sb.String()

	return result
}
