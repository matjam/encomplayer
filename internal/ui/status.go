package ui

import (
	"fmt"
	"strings"
	"time"
)

const statusTTL = 4 * time.Second

// status is the transient message on the bottom line.
type status struct {
	text    string
	isError bool
	until   time.Time
}

func (s *status) infof(format string, args ...any) {
	s.text = strings.ToUpper(fmt.Sprintf(format, args...))
	s.isError = false
	s.until = time.Now().Add(statusTTL)
}

func (s *status) errorf(format string, args ...any) {
	s.text = fmt.Sprintf(format, args...)
	s.isError = true
	s.until = time.Now().Add(2 * statusTTL)
}

func (s *status) expire() {
	if s.text != "" && time.Now().After(s.until) {
		s.text = ""
	}
}
