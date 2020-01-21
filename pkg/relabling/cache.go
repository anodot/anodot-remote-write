package relabling

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const AnodotPodNameLabel string = "anodot.com/podName"

type PodData struct {
	Name   string
	Labels map[string]string
}

func (p *PodData) AnodotPodName() string {
	return p.Labels[AnodotPodNameLabel]
}

type CacheKey string

func NewKey(namespace, podName string) CacheKey {
	return CacheKey(fmt.Sprintf("%s|%s", namespace, podName))
}

func (k CacheKey) GetPodNameAndNamespace() (namespace, podName string) {
	s := strings.Split(string(k), "|")
	return s[0], s[1]
}

type PodCache struct {
	mu sync.RWMutex
	//Namespace|podname CacheKey example: kube-system|nginx-123123-123123
	Data map[CacheKey]*PodData
}

func NewCache() *PodCache {
	return &PodCache{
		mu:   sync.RWMutex{},
		Data: map[CacheKey]*PodData{},
	}
}

type SaveEntry struct {
	Name      string
	Labels    map[string]string
	Namespace string
}

type SearchEntry struct {
	PodName   string
	Namespace string
}

func (s SearchEntry) String() string {
	return fmt.Sprintf("%s|%s", s.Namespace, s.PodName)
}

func (p *PodCache) Store(e SaveEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Data[NewKey(e.Namespace, e.Name)] = &PodData{
		Labels: e.Labels,
		Name:   e.Name,
	}
}

func (p *PodCache) Replace(e map[CacheKey]*PodData) {
	p.mu.Lock()
	defer p.mu.Unlock()

	newMap := make(map[CacheKey]*PodData, len(e))
	for k, v := range e {
		newMap[k] = v
	}

	p.Data = newMap
}

func (p *PodCache) Lookup(e SearchEntry) *PodData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Data[NewKey(e.Namespace, e.PodName)]
}

func (p *PodCache) LookupAllNamespaces(podname string) *PodData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	values := make([]*PodData, 0)
	for key, v := range p.Data {
		if strings.HasSuffix(string(key), fmt.Sprintf("|%s", podname)) {
			values = append(values, v)
		}
	}

	if len(values) != 1 {
		return nil
	}

	return values[0]
}

func (p *PodCache) Entries() map[CacheKey]*PodData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Data
}

func (p *PodCache) Delete(e SearchEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.Data, NewKey(e.Namespace, e.PodName))
}

func (p *PodCache) PrintEntries() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, _ := json.Marshal(p.Data)
	fmt.Println(string(data))
}

func (p *PodCache) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.Data)
}
