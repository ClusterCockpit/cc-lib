package util

import (
	"encoding/json"
	"testing"
)

func TestSelectorReuseFreshVariants(t *testing.T) {
	for _, raw := range []string{`"cpu0"`, `"*"`, `["cpu0","cpu1"]`, `[]`} {
		var value SelectorElement
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(&value)
		if err != nil || string(encoded) != raw {
			t.Fatalf("%s -> %s, %v", raw, encoded, err)
		}
	}
	for _, raw := range []string{`null`, `17`, `true`, `["cpu0",1]`} {
		var value SelectorElement
		if err := json.Unmarshal([]byte(raw), &value); err == nil {
			t.Fatalf("unsupported selector %s accepted", raw)
		}
	}
}

func TestSelectorReuseLatestValue(t *testing.T) {
	variants := []string{`"cpu0"`, `"*"`, `["cpu1"]`, `[]`}
	for _, initial := range variants {
		for _, latest := range variants {
			var value SelectorElement
			for _, raw := range []string{initial, latest} {
				if err := json.Unmarshal([]byte(raw), &value); err != nil {
					t.Fatal(err)
				}
			}
			encoded, err := json.Marshal(&value)
			if err != nil || string(encoded) != latest {
				t.Errorf("%s then %s: got %s, err=%v", initial, latest, encoded, err)
			}
		}
	}
}
