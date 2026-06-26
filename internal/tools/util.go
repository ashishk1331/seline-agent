package tools

import "sort"

// topThree returns the three most frequent values, most-common first. It is the
// Go port of utils.top_three (collections.Counter.most_common(3)).
func topThree(texts []string) []string {
	counts := make(map[string]int)
	order := make([]string, 0, len(texts))
	for _, t := range texts {
		if _, seen := counts[t]; !seen {
			order = append(order, t)
		}
		counts[t]++
	}

	// stable sort: by count desc, ties keep first-seen order
	sort.SliceStable(order, func(i, j int) bool {
		return counts[order[i]] > counts[order[j]]
	})

	if len(order) > 3 {
		order = order[:3]
	}
	return order
}
