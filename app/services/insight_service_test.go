package services

import (
	"fmt"
	"testing"
)

var insightFilterCategories = map[string]bool{
	"Saham":       true,
	"SBSN":        true,
	"Logam Mulia": true,
}

func TestContohInsightItemsCoverEveryFilter(t *testing.T) {
	items := contohInsightItems()

	if len(items) != insightMaxItems {
		t.Fatalf("contohInsightItems() = %d items, want %d", len(items), insightMaxItems)
	}

	counts := make(map[string]int, len(insightFilterCategories))
	for _, item := range items {
		if !insightFilterCategories[item.Category] {
			t.Errorf("item %q has category %q, want one of Saham/SBSN/Logam Mulia", item.Title, item.Category)
		}

		counts[item.Category]++
	}

	for category := range insightFilterCategories {
		if counts[category] == 0 {
			t.Errorf("contohInsightItems() has no item for category %q", category)
		}
	}
}

func TestInsightFeedTopicsUseFilterCategories(t *testing.T) {
	for _, topic := range insightFeedTopics {
		if !insightFilterCategories[topic.Category] {
			t.Errorf("feed topic %q has category %q, want one of Saham/SBSN/Logam Mulia", topic.Query, topic.Category)
		}
	}
}

func TestMergeInsightItemsCapsAtMaxItems(t *testing.T) {
	items := make([]InsightItem, 0, insightMaxItems+5)
	for index := 0; index < insightMaxItems+5; index++ {
		items = append(items, InsightItem{
			Title:    fmt.Sprintf("Berita uji %d", index),
			Category: "Saham",
			Date:     "2026-10-20",
		})
	}

	merged := mergeInsightItems(items)
	if len(merged) != insightMaxItems {
		t.Fatalf("mergeInsightItems() = %d items, want %d", len(merged), insightMaxItems)
	}
}

func TestInsightCoversEveryCategory(t *testing.T) {
	complete := contohInsightItems()
	if !insightCoversEveryCategory(complete) {
		t.Fatal("contohInsightItems() should cover every filter category")
	}

	partial := make([]InsightItem, 0, len(complete))
	for _, item := range complete {
		if item.Category != "Saham" {
			partial = append(partial, item)
		}
	}

	if insightCoversEveryCategory(partial) {
		t.Fatal("insightCoversEveryCategory() = true for a list without Saham items")
	}
}
