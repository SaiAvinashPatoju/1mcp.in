package envtmpl

import (
	"reflect"
	"testing"
)

func TestExpandSimple(t *testing.T) {
	got, miss := Expand("hello ${NAME}!", map[string]string{"NAME": "world"})
	if got != "hello world!" || len(miss) != 0 {
		t.Fatalf("got %q miss %v", got, miss)
	}
}

func TestExpandMissing(t *testing.T) {
	_, miss := Expand("${A}-${B}", map[string]string{"A": "x"})
	if !reflect.DeepEqual(miss, []string{"B"}) {
		t.Fatalf("miss %v", miss)
	}
}

func TestExpandIgnoresInvalid(t *testing.T) {
	got, _ := Expand("${1BAD} stays", map[string]string{"1BAD": "x"})
	if got != "${1BAD} stays" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandAllDedup(t *testing.T) {
	_, miss := ExpandAll([]string{"${X}", "${X}-${Y}"}, nil)
	if len(miss) != 2 {
		t.Fatalf("expected 2 unique missing, got %v", miss)
	}
}
