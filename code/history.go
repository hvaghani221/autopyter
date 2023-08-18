package code

import (
	"sync"
)

type History struct {
	Items  []HistoryItem
	mu     sync.RWMutex
	lastID int64
}

type HistoryItem struct {
	Code string
	ID   int64
}

func (hi HistoryItem) GetID() int64 {
	return hi.ID
}

func NewHistory() *History {
	return &History{
		Items: make([]HistoryItem, 0),
		mu:    sync.RWMutex{},
	}
}

func (h *History) Add(item string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Items = append(h.Items, HistoryItem{
		Code: item,
		ID:   h.lastID,
	})
	h.lastID++
}

func (h *History) Get(id int64) (HistoryItem, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, item := range h.Items {
		if item.ID == id {
			return item, true
		}
	}
	return HistoryItem{}, false
}

func (h *History) Update(id int64, item HistoryItem) {
	h.mu.Lock()
	defer h.mu.Unlock()

	idx := -1
	for i, item := range h.Items {
		if item.ID != id {
			continue
		}
		idx = i
		break
	}
	if idx == -1 {
		panic("item not found")
	}
	h.Items[idx] = item
}

func (h *History) List() []HistoryItem {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := make([]HistoryItem, len(h.Items))
	copy(items, h.Items)
	return items
}

func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Items = make([]HistoryItem, 0)
}

func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Items)
}

func (h *History) Remove(id int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, item := range h.Items {
		if item.ID == id {
			h.Items = append(h.Items[:i], h.Items[i+1:]...)
			return true
		}
	}
	return false
}
