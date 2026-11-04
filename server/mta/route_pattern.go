package mta

import (
	"regexp"
	"sort"
	"strings"
)

var splitServicePattern = regexp.MustCompile(`(?i)\b(?:runs|operates)\s+in\s+(?:two|three|four|\d+)\s+sections?\b`)
var sectionPattern = regexp.MustCompile(`(?i)^\s*\d+[.)]?\s+.*?\bbetween\s+(.+?)\s+and\s+(.+?)(?:,|\s+stopping\b|\s+every\b|$)`)
var stationNamePattern = regexp.MustCompile(`[^a-z0-9]+`)

func isSplitServiceAlert(alert RouteAlert) bool {
	return splitServicePattern.MatchString(alert.Header + "\n" + alert.Description)
}
func normalizedStationName(value string) string {
	return strings.TrimSpace(stationNamePattern.ReplaceAllString(strings.ToLower(strings.NewReplacer("–", "-", "—", "-").Replace(value)), " "))
}
func sectionEndpointID(value string, stops []StopObject) string {
	target := normalizedStationName(value)
	if target == "" {
		return ""
	}
	for _, stop := range stops {
		if normalizedStationName(stop.Name) == target {
			return stop.ID
		}
	}
	matches := []string{}
	for _, stop := range stops {
		name := normalizedStationName(stop.Name)
		if strings.HasPrefix(name, target) || strings.HasPrefix(target, name) {
			matches = append(matches, stop.ID)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}
func alertSectionStopIDs(alerts []RouteAlert, stops []StopObject) map[string]struct{} {
	active := map[string]struct{}{}
	for _, alert := range alerts {
		if !isSplitServiceAlert(alert) {
			continue
		}
		for _, line := range strings.Split(strings.ReplaceAll(alert.Header+"\n"+alert.Description, "\r\n", "\n"), "\n") {
			section := sectionPattern.FindStringSubmatch(line)
			if len(section) < 3 {
				continue
			}
			first, last := sectionEndpointID(section[1], stops), sectionEndpointID(section[2], stops)
			start, end := -1, -1
			for index, stop := range stops {
				if start < 0 && stop.ID == first {
					start = index
				}
				if end < 0 && stop.ID == last {
					end = index
				}
			}
			if start < 0 || end < 0 {
				continue
			}
			if start > end {
				start, end = end, start
			}
			for _, stop := range stops[start : end+1] {
				active[stop.ID] = struct{}{}
			}
		}
	}
	return active
}

type stopPattern struct {
	StopIDs []string
	Count   int
}
type terminalBranch struct {
	JunctionID, TerminalID, Placement string
	StopIDs                           []string
}

func containsContiguousStops(candidate, stops []string) bool {
	if len(stops) == 0 || len(candidate) < len(stops) {
		return false
	}
	for start := 0; start <= len(candidate)-len(stops); start++ {
		matched := true
		for index, id := range stops {
			if candidate[start+index] != id {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
func extendShortTurnPattern(pattern *stopPattern, available []stopPattern) *stopPattern {
	if pattern == nil {
		return nil
	}
	candidates := []stopPattern{}
	for _, candidate := range available {
		if candidate.Count >= pattern.Count && containsContiguousStops(candidate.StopIDs, pattern.StopIDs) {
			candidates = append(candidates, candidate)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if len(candidates[i].StopIDs) == len(candidates[j].StopIDs) {
			return candidates[i].Count > candidates[j].Count
		}
		return len(candidates[i].StopIDs) > len(candidates[j].StopIDs)
	})
	if len(candidates) > 0 {
		return &candidates[0]
	}
	return pattern
}
func mostCompletePattern(patterns []stopPattern) *stopPattern {
	grouped := map[string]stopPattern{}
	order := []string{}
	for _, pattern := range patterns {
		if len(pattern.StopIDs) == 0 {
			continue
		}
		key := stringsJoin(pattern.StopIDs, ",")
		value, found := grouped[key]
		if !found {
			value = stopPattern{StopIDs: pattern.StopIDs}
			order = append(order, key)
		}
		value.Count++
		grouped[key] = value
	}
	values := make([]stopPattern, 0, len(order))
	for _, key := range order {
		values = append(values, grouped[key])
	}
	sort.SliceStable(values, func(i, j int) bool {
		if len(values[i].StopIDs) == len(values[j].StopIDs) {
			return values[i].Count > values[j].Count
		}
		return len(values[i].StopIDs) > len(values[j].StopIDs)
	})
	if len(values) == 0 {
		return nil
	}
	return &values[0]
}
func countActiveRuns(stops []string, active map[string]struct{}) int {
	runs := 0
	was := false
	for _, id := range stops {
		_, is := active[id]
		if is && !was {
			runs++
		}
		was = is
	}
	return runs
}
func applyRealtimeDetour(base []string, patterns []stopPattern) []string {
	index := map[string]int{}
	for i, id := range base {
		index[id] = i
	}
	type detour struct {
		start, end, score int
		replacement       []string
	}
	var best *detour
	for _, pattern := range patterns {
		stops := append([]string(nil), pattern.StopIDs...)
		shared := []string{}
		for _, id := range stops {
			if _, ok := index[id]; ok {
				shared = append(shared, id)
			}
		}
		if len(shared) < 2 {
			continue
		}
		if index[shared[0]] > index[shared[len(shared)-1]] {
			reverseStrings(stops)
		}
		for i := 1; i < len(stops)-1; i++ {
			if _, ok := index[stops[i]]; ok {
				continue
			}
			start := i
			for i < len(stops) {
				if _, ok := index[stops[i]]; ok {
					break
				}
				i++
			}
			if i == len(stops) {
				break
			}
			startID, endID := stops[start-1], stops[i]
			startIndex, startOK := index[startID]
			endIndex, endOK := index[endID]
			if !startOK || !endOK || startIndex >= endIndex {
				continue
			}
			replacement := append([]string(nil), stops[start-1:i+1]...)
			score := (endIndex-startIndex)*100 + len(replacement)
			if best == nil || score > best.score {
				best = &detour{start: startIndex, end: endIndex, replacement: replacement, score: score}
			}
		}
	}
	if best == nil {
		return base
	}
	result := append([]string(nil), base[:best.start]...)
	result = append(result, best.replacement...)
	result = append(result, base[best.end+1:]...)
	return result
}
func offRouteTerminalBranches(base []string, patterns []stopPattern) []terminalBranch {
	baseSet := map[string]struct{}{}
	baseIndex := map[string]int{}
	for i, id := range base {
		baseSet[id] = struct{}{}
		baseIndex[id] = i
	}
	branches := map[string]terminalBranch{}
	save := func(junction string, stops []string, placement string) {
		if junction == "" || len(stops) == 0 {
			return
		}
		terminal := stops[len(stops)-1]
		if placement == "before" {
			terminal = stops[0]
		}
		key := junction + ":" + terminal
		existing, found := branches[key]
		if !found || len(stops) > len(existing.StopIDs) {
			branches[key] = terminalBranch{junction, terminal, placement, append([]string(nil), stops...)}
		}
	}
	for _, pattern := range patterns {
		stops := pattern.StopIDs
		first := -1
		last := -1
		for i, id := range stops {
			if _, ok := baseSet[id]; ok {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 {
			continue
		}
		matched := []int{}
		for _, id := range stops[first : last+1] {
			if value, ok := baseIndex[id]; ok {
				matched = append(matched, value)
			}
		}
		forward := len(matched) < 2 || matched[len(matched)-1] >= matched[0]
		prefix := append([]string(nil), stops[:first]...)
		suffix := append([]string(nil), stops[last+1:]...)
		if !forward {
			reverseStrings(prefix)
			reverseStrings(suffix)
		}
		if forward {
			save(stops[first], prefix, "before")
			save(stops[last], suffix, "after")
		} else {
			save(stops[first], prefix, "after")
			save(stops[last], suffix, "before")
		}
	}
	values := []terminalBranch{}
	for _, branch := range branches {
		values = append(values, branch)
	}
	result := []terminalBranch{}
	for _, branch := range values {
		hidden := false
		for _, candidate := range values {
			if candidate.JunctionID == branch.JunctionID && candidate.Placement == branch.Placement && len(candidate.StopIDs) > len(branch.StopIDs) && containsContiguousStops(candidate.StopIDs, branch.StopIDs) {
				hidden = true
				break
			}
		}
		if !hidden {
			result = append(result, branch)
		}
	}
	return result
}
func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
func stringsJoin(values []string, separator string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += separator
		}
		result += value
	}
	return result
}
