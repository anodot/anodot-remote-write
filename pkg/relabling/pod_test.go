package relabling

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestJsonMarshal(t *testing.T) {
	includedPods := NewCache()
	excluded := NewCache()

	excluded.Store(SaveEntry{
		Name:        "exluded",
		ChangedName: "name-1",
		Namespace:   "kube-system",
	})

	includedPods.Store(SaveEntry{
		Name:        "included",
		ChangedName: "name-2",
		Namespace:   "kube-system",
	})
	mapping := PodsMapping{
		WhitelistedPods: includedPods,
		ExcludedPods:    excluded,
	}

	marshal, err := json.Marshal(mapping)
	if err != nil {
		t.Fatal(err.Error())
	}

	s := `{"WhitelistedPods":{"Data":{"kube-system|included":"name-2"}},"ExcludedPods":{"Data":{"kube-system|exluded":"name-1"}}}`
	if !reflect.DeepEqual(marshal, []byte(s)) {
		t.Fatal(fmt.Sprintf("Wrong json for PodsMapping \n got: %s\n want: %s", string(marshal), s))
	}
}
