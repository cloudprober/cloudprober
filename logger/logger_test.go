package logger

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarSet(t *testing.T) {
	varName := "TEST_VAR"

	testRows := []struct {
		v        string
		expected bool
	}{
		{"1", true},
		{"yes", true},
		{"not_set", false},
		{"no", false},
		{"false", false},
	}

	for _, row := range testRows {
		t.Run(fmt.Sprintf("Val: %s, should be set: %v", row.v, row.expected), func(t *testing.T) {
			os.Unsetenv(varName)
			if row.v != "not_set" {
				os.Setenv(varName, row.v)
			}

			got := envVarSet(varName)
			if got != row.expected {
				t.Errorf("Variable set: got=%v, expected=%v", got, row.expected)
			}
		})
	}
}

func TestWithLabels(t *testing.T) {
	panicOnErr := func(l *Logger, err error) *Logger {
		if err != nil {
			panic(err)
		}
		return l
	}
	tests := []struct {
		l          *Logger
		wantLabels map[string]string
	}{
		{
			l:          panicOnErr(New(context.Background(), "newWithLabels", WithLabels(map[string]string{"k1": "v1"}))),
			wantLabels: map[string]string{"k1": "v1"},
		},
		{
			l:          panicOnErr(New(context.Background(), "newWithLabels", WithLabels(map[string]string{"k1": "v1"}), WithLabels(map[string]string{"k1": "v2"}))),
			wantLabels: map[string]string{"k1": "v2"},
		},
		{
			l: func() *Logger {
				l := &Logger{}
				WithLabels(map[string]string{"k1": "v1", "k3": "v3"})(l)
				return l
			}(),
			wantLabels: map[string]string{"k1": "v1", "k3": "v3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.l.name, func(t *testing.T) {
			assert.Equal(t, tt.wantLabels, tt.l.labels)
		})
	}
}
