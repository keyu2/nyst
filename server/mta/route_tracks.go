package mta

func trackEdge(first, second string) string {
	if first > second {
		first, second = second, first
	}
	return first + "|" + second
}

func applyTrackService(routeID string, stops []RouteStop, branches []RouteBranch, patterns []stopPattern, known bool) {
	if !known {
		return
	}
	graph := map[string][]string{}
	connect := func(first, second string) {
		if first == "" || second == "" || first == second {
			return
		}
		graph[first] = append(graph[first], second)
		graph[second] = append(graph[second], first)
	}
	junction := func(branch RouteBranch) (string, bool) {
		if routeID == "5" && branch.ID == "nereid" {
			return "213", true
		}
		if routeID == "A" {
			if branch.ID == "rockaway-park" {
				return "H04", false
			}
			return "A61", false
		}
		return branch.JunctionID, branch.Placement == "before"
	}
	addStops := func(items []RouteStop) {
		for index := 1; index < len(items); index++ {
			connect(items[index-1].ID, items[index].ID)
		}
	}
	branchStopIDs := map[string]bool{}
	for _, branch := range branches {
		for _, stop := range branch.Stops {
			branchStopIDs[stop.ID] = true
		}
	}
	trunk := []RouteStop{}
	for _, stop := range stops {
		if !branchStopIDs[stop.ID] {
			trunk = append(trunk, stop)
		}
	}
	addStops(trunk)
	for _, branch := range branches {
		addStops(branch.Stops)
		if len(branch.Stops) == 0 {
			continue
		}
		id, before := junction(branch)
		if before {
			connect(branch.Stops[len(branch.Stops)-1].ID, id)
		} else {
			connect(id, branch.Stops[0].ID)
		}
	}
	active := map[string]bool{}
	for _, pattern := range patterns {
		for index := 1; index < len(pattern.StopIDs); index++ {
			start, end := pattern.StopIDs[index-1], pattern.StopIDs[index]
			parents := map[string]string{start: ""}
			queue := []string{start}
			for position := 0; position < len(queue); position++ {
				current := queue[position]
				if current == end {
					break
				}
				for _, next := range graph[current] {
					if _, seen := parents[next]; !seen {
						parents[next] = current
						queue = append(queue, next)
					}
				}
			}
			if _, found := parents[end]; !found {
				continue
			}
			for current := end; current != start; current = parents[current] {
				active[trackEdge(current, parents[current])] = true
			}
		}
	}
	mark := func(items []RouteStop, previous, next string) {
		for index := range items {
			above, below := previous, next
			if index > 0 {
				above = items[index-1].ID
			}
			if index+1 < len(items) {
				below = items[index+1].ID
			}
			aboveActive := active[trackEdge(above, items[index].ID)]
			belowActive := active[trackEdge(items[index].ID, below)]
			items[index].TrackServiceAbove = &aboveActive
			items[index].TrackServiceBelow = &belowActive
		}
	}
	mark(stops, "", "")
	for _, branch := range branches {
		id, before := junction(branch)
		if before {
			mark(branch.Stops, "", id)
		} else {
			mark(branch.Stops, id, "")
		}
	}
}
