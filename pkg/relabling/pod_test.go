package relabling

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestMarshar(t *testing.T) {
	includedPods := NewCache()
	excluded := NewCache()

	excluded.Store(SaveEntry{
		Name:      "exluded",
		Labels:    map[string]string{"1": "2"},
		Namespace: "kube-system",
	})

	includedPods.Store(SaveEntry{
		Name:      "included",
		Labels:    map[string]string{"1": "2"},
		Namespace: "kube-system",
	})
	mapping := PodsMapping{
		WhitelistedPods: includedPods,
		ExcludedPods:    excluded,
	}

	marshal, err := json.Marshal(mapping)
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Println(string(marshal))
}
