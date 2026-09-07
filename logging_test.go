package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// log.Println must end up in the file as a JSON line once setupLog ran.
func TestSetupLogRoutesStdLog(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "x.log") // dir is created
	if err := setupLog(p); err != nil {
		t.Fatal(err)
	}
	log.Println("hello", 42)
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"msg":"hello 42"`) {
		t.Fatalf("log file: %q", b)
	}
}
