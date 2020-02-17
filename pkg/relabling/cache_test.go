package relabling

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestCacheLookUp(t *testing.T) {
	cache := NewCache()

	cache.Store(SaveEntry{
		Name:        "test-pod-xfxf",
		ChangedName: "test-pod-0",
		Namespace:   "system",
	})

	cache.Store(SaveEntry{
		Name:        "public-pod-xfxf",
		ChangedName: "public-pod-1",
		Namespace:   "public",
	})

	changedName := cache.Lookup(SearchEntry{
		PodName:   "public-pod-xfxf",
		Namespace: "public",
	})

	if changedName != "public-pod-1" {
		t.Fatal(fmt.Sprintf("Wrong json for PodsMapping \n got: %s\n want: %s", changedName, "public-pod-1"))
	}
}

func TestCacheLookUpAllNamespaces(t *testing.T) {
	cache := NewCache()

	cache.Store(SaveEntry{
		Name:        "test-pod-xfxf",
		ChangedName: "test-pod-0",
		Namespace:   "system",
	})

	cache.Store(SaveEntry{
		Name:        "public-pod-xfxf",
		ChangedName: "public-pod-1",
		Namespace:   "public",
	})

	cache.Store(SaveEntry{
		Name:        "public-pod-xfxf",
		ChangedName: "system-pod-3",
		Namespace:   "system",
	})

	changedName := cache.LookupAllNamespaces("test-pod-xfxf")
	if changedName != "test-pod-0" {
		t.Fatal(fmt.Sprintf("wrong pods changedName \n got: %s\n want: %s", changedName, "public-pod-1"))
	}

	//if same name in different namespace, "" should be returned
	changedName = cache.LookupAllNamespaces("public-pod-xfxf")
	if changedName != "" {
		t.Fatal(fmt.Sprintf("wrong pods changedName \n got: %s\n want: %s", changedName, ""))
	}
}

func TestCacheReplace(t *testing.T) {
	cache := NewCache()

	cache.Store(SaveEntry{
		Name:        "test-pod-xfxf",
		ChangedName: "test-pod-0",
		Namespace:   "system",
	})

	newData := make(map[CacheKey]string)
	newData[NewKey("public", "test-pod-xffffr")] = "test-pod-0"

	cache.Replace(newData)

	if !reflect.DeepEqual(cache.Data, newData) {
		t.Fatal(fmt.Sprintf("wrong pod cache data \n got: %s\n want: %s", cache.Data, newData))
	}
}

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

func TestCacheDelete(t *testing.T) {
	cache := NewCache()

	cache.Store(SaveEntry{
		Name:        "test-pod-xfxf",
		ChangedName: "test-pod-0",
		Namespace:   "system",
	})

	searchEntry := SearchEntry{
		PodName:   "test-pod-xfxf",
		Namespace: "system",
	}
	cache.Delete(searchEntry)

	lookupEntry := cache.Lookup(searchEntry)
	if lookupEntry != "" {
		t.Fatal("entry should be removed")
	}

}

func TestCacheKey(t *testing.T) {
	cacheKey := NewKey("system", "test-pod-xfxf")

	namespace, podName := cacheKey.GetPodNameAndNamespace()

	if namespace != "system" {
		t.Fatal(fmt.Sprintf("Wrong namespace name \n got: %s\n want: %s", namespace, "system"))
	}

	if podName != "test-pod-xfxf" {
		t.Fatal(fmt.Sprintf("Wrong podName \n got: %s\n want: %s", podName, "test-pod-xfxf"))
	}

	if cacheKey.AnodotPodName() != "test-pod-xfxf" {
		t.Fatal(fmt.Sprintf("Wrong AnodotPodName \n got: %s\n want: %s", podName, "test-pod-xfxf"))
	}
}
