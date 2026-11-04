package mta

import (
	"context"
	"sort"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type RouteStop struct {
	StopObject
	TransferRoutes    []string `json:"transferRoutes"`
	HasService        *bool    `json:"hasService"`
	TrackServiceAbove *bool    `json:"trackServiceAbove,omitempty"`
	TrackServiceBelow *bool    `json:"trackServiceBelow,omitempty"`
}
type RouteBranch struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Placement  string      `json:"placement,omitempty"`
	JunctionID string      `json:"junctionId,omitempty"`
	Temporary  bool        `json:"temporary,omitempty"`
	Stops      []RouteStop `json:"stops"`
}
type ServiceMap struct {
	ConfigID string      `json:"configId"`
	Stops    []RouteStop `json:"stops"`
}
type RouteResponse struct {
	ID               string        `json:"id"`
	Color            string        `json:"color"`
	ServiceMaps      []ServiceMap  `json:"serviceMaps"`
	Branches         []RouteBranch `json:"branches"`
	Alerts           []RouteAlert  `json:"alerts"`
	ServiceAvailable *bool         `json:"serviceAvailable"`
}

func (s *TransitService) Route(ctx context.Context, id string, now time.Time, alerts ...RouteAlert) (RouteResponse, error) {
	data, err := s.static.load(ctx)
	if err != nil {
		return RouteResponse{}, err
	}
	entities, realtimeErr := s.realtime.Entities(ctx)
	metadata := s.metadata.Get(ctx)
	family := routeFamily(id)
	activeServices := activeServiceIDs(data, now)
	candidates := []staticTrip{}
	scheduled := []staticTrip{}
	for _, trip := range data.Trips {
		if normalizeRouteID(trip.RouteID) != id {
			continue
		}
		candidates = append(candidates, trip)
		if _, ok := activeServices[trip.ServiceID]; ok {
			scheduled = append(scheduled, trip)
		}
	}
	near := tripsNearMinute(scheduled, data.StopTimesByTrip, currentNewYorkSecond(now)/60, 120)
	patternTrips := near
	if len(patternTrips) == 0 {
		patternTrips = scheduled
	}
	if len(patternTrips) == 0 {
		patternTrips = candidates
	}
	directionCounts := map[string]int{}
	for _, trip := range patternTrips {
		directionCounts[trip.DirectionID]++
	}
	preferred := ""
	best := -1
	for direction, count := range directionCounts {
		if count > best || (count == best && direction < preferred) {
			preferred, best = direction, count
		}
	}
	patterns := collectPatterns(patternTrips, data.StopTimesByTrip, preferred)
	normal := mostFrequentPattern(patterns)
	canonical := collectPatterns(candidates, data.StopTimesByTrip, preferred)
	hasSplitService := false
	for _, alert := range alerts {
		if isSplitServiceAlert(alert) {
			hasSplitService = true
			break
		}
	}
	if hasSplitService {
		if full := mostFrequentPattern(canonical); full != nil {
			normal = full
		}
	} else {
		normal = extendShortTurnPattern(normal, canonical)
	}
	normalIDs := []string{}
	if normal != nil {
		normalIDs = append(normalIDs, normal.StopIDs...)
	}
	realtimePatterns := []stopPattern{}
	activeStops := map[string]struct{}{}
	nowUnix := now.Unix()
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		if routeFamily(update.GetTrip().GetRouteId()) != family || int32(update.GetTrip().GetScheduleRelationship()) == 3 {
			continue
		}
		upcoming := false
		ids := []string{}
		for _, item := range update.GetStopTimeUpdate() {
			if eventTimestamp(item) >= nowUnix {
				upcoming = true
			}
			if isPublicStop(item.GetStopId(), data) {
				base := stationStopID(item.GetStopId())
				if len(ids) == 0 || ids[len(ids)-1] != base {
					ids = append(ids, base)
				}
			}
		}
		if upcoming && len(ids) >= 2 {
			realtimePatterns = append(realtimePatterns, stopPattern{StopIDs: ids})
			for _, stop := range ids {
				activeStops[stop] = struct{}{}
			}
		}
	}
	canonicalStops := []StopObject{}
	for _, id := range normalIDs {
		canonicalStops = append(canonicalStops, StopObject{ID: id, Name: stopObject(id, data, metadata).Name})
	}
	for id := range alertSectionStopIDs(alerts, canonicalStops) {
		activeStops[id] = struct{}{}
	}
	detoured := applyRealtimeDetour(normalIDs, realtimePatterns)
	detourChanged := !sameStrings(detoured, normalIDs)
	normalIDs = detoured
	complete := mostCompletePattern(realtimePatterns)
	runs := countActiveRuns(normalIDs, activeStops)
	if complete != nil && runs <= 1 && !hasSplitService && !detourChanged {
		normalIDs = complete.StopIDs
	}
	temporary := []terminalBranch{}
	if runs > 1 && id != "A" && id != "5" {
		temporary = offRouteTerminalBranches(normalIDs, realtimePatterns)
	}
	if id == "SI" && len(data.SIRLocalStopIDs) > 0 {
		normalIDs = append([]string(nil), data.SIRLocalStopIDs...)
		if normalIDs[0] != "S31" {
			reverseStrings(normalIDs)
		}
	}
	routesByStop := realtimeRoutesByStop(entities, now)
	routeStop := func(stopID string) RouteStop {
		object := stopObject(stopID, data, metadata)
		transfer := interchangeRoutes(stopID, data, id, routesByStop)
		var service *bool
		if realtimeErr == nil {
			value := false
			if _, ok := activeStops[stationStopID(stopID)]; ok {
				value = true
			}
			service = &value
		}
		return RouteStop{StopObject: object, TransferRoutes: transfer, HasService: service}
	}
	branches := []RouteBranch{}
	if id == "A" {
		normalIDs, branches = buildARouteBranches(candidates, scheduled, data, normalIDs, realtimePatterns, entities, now, realtimeErr == nil, routeStop)
	}
	if id == "5" {
		if len(normalIDs) > 0 && normalIDs[0] != "501" {
			reverseStrings(normalIDs)
		}
		nereid := []string{"204", "205", "206", "207", "208", "209", "210", "211", "212"}
		if hasServiceAtStops(routesByStop, nereid, "5") {
			branch := RouteBranch{ID: "nereid", Name: "Nereid Av"}
			for _, stop := range nereid {
				branch.Stops = append(branch.Stops, routeStop(stop))
			}
			branches = append(branches, branch)
		}
	}
	for _, branch := range temporary {
		item := RouteBranch{ID: "temporary-" + branch.JunctionID + "-" + branch.TerminalID, Name: stopObject(branch.TerminalID, data, metadata).Name, JunctionID: branch.JunctionID, Placement: branch.Placement, Temporary: true}
		for _, stop := range branch.StopIDs {
			item.Stops = append(item.Stops, routeStop(stop))
		}
		branches = append(branches, item)
	}
	stops := make([]RouteStop, 0, len(normalIDs))
	for _, stop := range normalIDs {
		stops = append(stops, routeStop(stop))
	}
	applyTrackService(id, stops, branches, realtimePatterns, realtimeErr == nil)
	var available *bool
	if realtimeErr == nil {
		value := false
		for _, entity := range entities {
			update := entity.GetTripUpdate()
			if normalizeRouteID(update.GetTrip().GetRouteId()) != normalizeRouteID(id) || int32(update.GetTrip().GetScheduleRelationship()) == 3 {
				continue
			}
			for _, stop := range update.GetStopTimeUpdate() {
				if timestamp := eventTimestamp(stop); timestamp > 0 && timestamp >= now.Unix()-60 {
					value = true
					break
				}
			}
			if value {
				break
			}
		}
		available = &value
	}
	return RouteResponse{ID: id, Color: routeColors[id], ServiceMaps: []ServiceMap{{ConfigID: "alltimes", Stops: stops}}, Branches: branches, Alerts: append([]RouteAlert{}, alerts...), ServiceAvailable: available}, nil
}

func collectPatterns(trips []staticTrip, times map[string][]stopTime, direction string) []stopPattern {
	grouped := map[string]stopPattern{}
	order := []string{}
	for _, trip := range trips {
		if trip.DirectionID != direction {
			continue
		}
		ids := []string{}
		for _, item := range times[trip.ID] {
			ids = append(ids, stationStopID(item.StopID))
		}
		if len(ids) == 0 {
			continue
		}
		key := stringsJoin(ids, ",")
		value, ok := grouped[key]
		if !ok {
			value = stopPattern{StopIDs: ids}
			order = append(order, key)
		}
		value.Count++
		grouped[key] = value
	}
	result := make([]stopPattern, 0, len(order))
	for _, key := range order {
		result = append(result, grouped[key])
	}
	return result
}
func mostFrequentPattern(patterns []stopPattern) *stopPattern {
	sort.SliceStable(patterns, func(i, j int) bool {
		if patterns[i].Count == patterns[j].Count {
			return len(patterns[i].StopIDs) > len(patterns[j].StopIDs)
		}
		return patterns[i].Count > patterns[j].Count
	})
	if len(patterns) == 0 {
		return nil
	}
	return &patterns[0]
}
func tripsNearMinute(trips []staticTrip, times map[string][]stopTime, now, window int) []staticTrip {
	result := []staticTrip{}
	for _, trip := range trips {
		items := times[trip.ID]
		if len(items) == 0 {
			continue
		}
		second := gtfsSecond(firstTime(items[0]))
		if second < 0 {
			continue
		}
		minute := second / 60
		distance := abs(minute - now)
		next := abs(minute - (now + 1440))
		if next < distance {
			distance = next
		}
		if distance <= window {
			result = append(result, trip)
		}
	}
	return result
}
func interchangeRoutes(stop string, data *staticData, current string, routes map[string]map[string]struct{}) []string {
	values := []string{}
	for connected := range connectedStopIDs(stop, data) {
		for route := range routes[connected] {
			if routeFamily(route) != routeFamily(current) {
				values = append(values, routeFamily(route))
			}
		}
	}
	values = uniqueStrings(values)
	sort.Slice(values, func(i, j int) bool { return naturalLess(values[i], values[j]) })
	return values
}
func hasServiceAtStops(routes map[string]map[string]struct{}, stops []string, route string) bool {
	for _, stop := range stops {
		if _, ok := routes[stop][route]; ok {
			return true
		}
	}
	return false
}
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type aBranchDefinition struct{ ID, Name, Terminal string }

var aBranchDefinitions = []aBranchDefinition{
	{"lefferts", "Lefferts Blvd", "A65"},
	{"far-rockaway", "Far Rockaway", "H11"},
	{"rockaway-park", "Rockaway Park", "H15"},
}

var rockawayParkStops = map[string]struct{}{"H12": {}, "H13": {}, "H14": {}, "H15": {}}

func overlayRealtimeStops(staticStops, realtimeStops []string) []string {
	staticIndex := map[string]int{}
	for index, stop := range staticStops {
		staticIndex[stop] = index
	}
	shared := []string{}
	for _, stop := range realtimeStops {
		if _, ok := staticIndex[stop]; ok {
			shared = append(shared, stop)
		}
	}
	if len(shared) < 2 {
		return append([]string(nil), staticStops...)
	}
	ordered := append([]string(nil), realtimeStops...)
	if staticIndex[shared[0]] > staticIndex[shared[len(shared)-1]] {
		reverseStrings(ordered)
	}
	firstRealtime, lastRealtime := -1, -1
	for index, stop := range ordered {
		if _, ok := staticIndex[stop]; ok {
			if firstRealtime < 0 {
				firstRealtime = index
			}
			lastRealtime = index
		}
	}
	firstStatic := staticIndex[ordered[firstRealtime]]
	lastStatic := staticIndex[ordered[lastRealtime]]
	if firstStatic >= lastStatic {
		return append([]string(nil), staticStops...)
	}
	result := append([]string(nil), staticStops[:firstStatic]...)
	result = append(result, ordered[firstRealtime:lastRealtime+1]...)
	result = append(result, staticStops[lastStatic+1:]...)
	return dedupeAdjacent(result)
}

func restoreActiveCanonicalStops(stops, canonical []string, active map[string]struct{}) []string {
	result := append([]string(nil), stops...)
	for index, stop := range canonical {
		if _, ok := active[stop]; !ok || contains(result, stop) {
			continue
		}
		next := ""
		for _, candidate := range canonical[index+1:] {
			if contains(result, candidate) {
				next = candidate
				break
			}
		}
		if next != "" {
			at := indexOf(result, next)
			result = append(result, "")
			copy(result[at+1:], result[at:])
			result[at] = stop
			continue
		}
		previous := ""
		for at := index - 1; at >= 0; at-- {
			if contains(result, canonical[at]) {
				previous = canonical[at]
				break
			}
		}
		at := len(result)
		if previous != "" {
			at = indexOf(result, previous) + 1
		}
		result = append(result, "")
		copy(result[at+1:], result[at:])
		result[at] = stop
	}
	return result
}

func completeATrunk(candidates []staticTrip, data *staticData) []string {
	patterns := [][]string{}
	for _, trip := range candidates {
		stops := stopIDsForTrip(trip, data)
		if !contains(stops, "A02") || !contains(stops, "A61") {
			continue
		}
		if indexOf(stops, "A02") > indexOf(stops, "A61") {
			reverseStrings(stops)
		}
		patterns = append(patterns, append([]string(nil), stops[:indexOf(stops, "A61")+1]...))
	}
	sort.Slice(patterns, func(i, j int) bool { return len(patterns[i]) > len(patterns[j]) })
	if len(patterns) == 0 {
		return nil
	}
	return patterns[0]
}

func operatesNearMinute(trip staticTrip, data *staticData, nowMinute int) bool {
	times := data.StopTimesByTrip[trip.ID]
	if len(times) == 0 {
		return false
	}
	first := gtfsSecond(firstTime(times[0])) / 60
	last := gtfsSecond(lastTime(times[len(times)-1])) / 60
	if first < 0 || last < 0 {
		return false
	}
	return (nowMinute >= first-30 && nowMinute <= last+30) || (nowMinute+1440 >= first-30 && nowMinute+1440 <= last+30)
}

func representativeABranches(candidates, scheduled []staticTrip, data *staticData, nowMinute int) []stopPattern {
	source := scheduled
	if len(source) == 0 {
		source = candidates
	}
	patterns := map[string]stopPattern{}
	idsByKey := map[string]string{}
	for _, trip := range source {
		stops := stopIDsForTrip(trip, data)
		if !contains(stops, "A61") || !contains(stops, "A02") {
			continue
		}
		if len(stops) > 0 && stops[0] != "A02" {
			reverseStrings(stops)
		}
		for _, definition := range aBranchDefinitions {
			if len(stops) == 0 || stops[len(stops)-1] != definition.Terminal || (definition.ID == "rockaway-park" && !operatesNearMinute(trip, data, nowMinute)) {
				continue
			}
			key := definition.ID + ":" + stringsJoin(stops, ",")
			pattern := patterns[key]
			pattern.StopIDs = stops
			pattern.Count++
			patterns[key] = pattern
			idsByKey[key] = definition.ID
		}
	}
	selected := []stopPattern{}
	for _, definition := range aBranchDefinitions {
		matches := []stopPattern{}
		for key, pattern := range patterns {
			if idsByKey[key] == definition.ID {
				matches = append(matches, pattern)
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			if matches[i].Count == matches[j].Count {
				return len(matches[i].StopIDs) > len(matches[j].StopIDs)
			}
			return matches[i].Count > matches[j].Count
		})
		if len(matches) > 0 {
			selected = append(selected, matches[0])
		}
	}
	return selected
}

func realtimeABranches(entities []*gtfs.FeedEntity, now int64) map[string]bool {
	active := map[string]bool{}
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		if routeFamily(update.GetTrip().GetRouteId()) != "A" || int32(update.GetTrip().GetScheduleRelationship()) == 3 {
			continue
		}
		for _, item := range update.GetStopTimeUpdate() {
			if eventTimestamp(item) < now {
				continue
			}
			switch stationStopID(item.GetStopId()) {
			case "A63", "A64", "A65":
				active["lefferts"] = true
			case "H01", "H02", "H03", "H04":
				active["rockaway-trunk"] = true
			case "H06", "H07", "H08", "H09", "H10", "H11":
				active["far-rockaway"] = true
			case "H12", "H13", "H14", "H15":
				active["rockaway-park"] = true
			}
		}
	}
	return active
}

func buildARouteBranches(candidates, scheduled []staticTrip, data *staticData, normal []string, realtimePatterns []stopPattern, entities []*gtfs.FeedEntity, now time.Time, realtimeAvailable bool, routeStop func(string) RouteStop) ([]string, []RouteBranch) {
	selected := representativeABranches(candidates, scheduled, data, currentNewYorkSecond(now)/60)
	activeBranches := realtimeABranches(entities, now.Unix())
	staticPattern := normal
	if len(selected) > 0 {
		staticPattern = selected[0].StopIDs
	}
	if realtimeAvailable {
		filtered := selected[:0]
		for _, pattern := range selected {
			for _, definition := range aBranchDefinitions {
				if len(pattern.StopIDs) > 0 && pattern.StopIDs[len(pattern.StopIDs)-1] == definition.Terminal && (activeBranches[definition.ID] || (definition.ID == "far-rockaway" && (activeBranches["rockaway-trunk"] || activeBranches["rockaway-park"]))) {
					filtered = append(filtered, pattern)
					break
				}
			}
		}
		selected = filtered
	}
	selectedIDs := make([]string, 0, len(selected))
	for _, pattern := range selected {
		for _, definition := range aBranchDefinitions {
			if len(pattern.StopIDs) > 0 && pattern.StopIDs[len(pattern.StopIDs)-1] == definition.Terminal {
				selectedIDs = append(selectedIDs, definition.ID)
			}
		}
	}
	if activeBranches["rockaway-park"] && !contains(selectedIDs, "rockaway-park") {
		selected = append(selected, stopPattern{StopIDs: []string{"H12", "H13", "H14", "H15"}, Count: 1})
		selectedIDs = append(selectedIDs, "rockaway-park")
	}
	junction := indexOf(staticPattern, "A61")
	if junction < 0 {
		return normal, nil
	}
	staticTrunk := append([]string(nil), staticPattern[:junction+1]...)
	active := map[string]struct{}{}
	for _, stop := range append(staticTrunk, normal...) {
		active[stop] = struct{}{}
	}
	for _, pattern := range realtimePatterns {
		for _, stop := range pattern.StopIDs {
			active[stop] = struct{}{}
		}
	}
	localTrunk := completeATrunk(candidates, data)
	combined := staticTrunk
	if len(localTrunk) > 0 {
		combined = nil
		for _, stop := range localTrunk {
			if _, ok := active[stop]; ok {
				combined = append(combined, stop)
			}
		}
	}
	trunk := restoreActiveCanonicalStops(overlayRealtimeStops(combined, normal), localTrunk, active)
	branches := []RouteBranch{}
	for index, pattern := range selected {
		definitionID := selectedIDs[index]
		definition := aBranchDefinitions[0]
		for _, candidate := range aBranchDefinitions {
			if candidate.ID == definitionID {
				definition = candidate
				break
			}
		}
		start := indexOf(pattern.StopIDs, "A61") + 1
		if definition.ID == "rockaway-park" {
			start = indexOf(pattern.StopIDs, "H12")
		}
		if start < 0 || start > len(pattern.StopIDs) {
			continue
		}
		stops := append([]string(nil), pattern.StopIDs[start:]...)
		if realtimeAvailable && definition.ID == "far-rockaway" && !activeBranches["far-rockaway"] {
			stops = stops[:indexOf(stops, "H04")+1]
		}
		if definition.ID == "far-rockaway" && !contains(stops, "H01") {
			stops = append([]string{"H01"}, stops...)
		}
		branch := RouteBranch{ID: definition.ID, Name: definition.Name}
		for _, stop := range stops {
			branch.Stops = append(branch.Stops, routeStop(stop))
		}
		branches = append(branches, branch)
	}
	return trunk, branches
}

func stopIDsForTrip(trip staticTrip, data *staticData) []string {
	stops := []string{}
	for _, item := range data.StopTimesByTrip[trip.ID] {
		stops = append(stops, stationStopID(item.StopID))
	}
	return stops
}

func dedupeAdjacent(values []string) []string {
	result := []string{}
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
func indexOf(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}
func contains(values []string, target string) bool { return indexOf(values, target) >= 0 }
