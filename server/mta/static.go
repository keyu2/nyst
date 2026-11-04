package mta

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const staticURL = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_subway.zip"

var stationComplexStopIDs = [][]string{
	{"718", "R09"}, {"132", "D19", "L02"}, {"635", "L03", "R20"},
	{"222", "415"}, {"414", "D11"}, {"112", "A09"}, {"719", "F09", "G22"},
	{"D17", "R17"}, {"F23", "R33"}, {"724", "D16"}, {"631", "723", "901"},
	{"127", "725", "902", "A27", "R16"}, {"630", "F11"}, {"629", "R11"},
	{"125", "A24"}, {"B16", "N04"}, {"710", "G14"}, {"235", "D24", "R31"},
	{"A31", "L01"}, {"637", "D21"}, {"232", "423", "R28"},
	{"A51", "J27", "L22"}, {"640", "M21"}, {"639", "M20", "Q01", "R23"},
	{"228", "A36", "E01", "R25"}, {"F15", "M18"}, {"239", "S04"},
	{"A45", "S01"}, {"229", "418", "A38", "M22"}, {"G29", "L10"},
	{"L17", "M08"}, {"142", "R27"}, {"A41", "R29"}, {"A12", "D13"},
	{"A32", "D20"},
}

type StationService struct {
	ID              string   `json:"id"`
	Routes          []string `json:"routes"`
	ScheduledRoutes []string `json:"scheduledRoutes"`
	RealtimeRoutes  []string `json:"realtimeRoutes"`
}

type Station struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Latitude  float64          `json:"latitude"`
	Longitude float64          `json:"longitude"`
	Services  []StationService `json:"services"`
}

type Bootstrap struct {
	Routes        []string                `json:"routes"`
	Stations      []Station               `json:"stations"`
	GeneratedAt   time.Time               `json:"generatedAt"`
	RouteAlerts   map[string][]RouteAlert `json:"routeAlerts"`
	RouteStatuses map[string]string       `json:"routeStatuses"`
}

type staticStop struct {
	Accessible                     bool
	ID, Name, Parent, LocationType string
	Latitude, Longitude            float64
}
type staticTrip struct {
	ID, RouteID, ServiceID, DirectionID, ShapeID string
}
type stopTime struct {
	StopID, ArrivalTime, DepartureTime string
	Sequence                           int
}
type calendarService struct {
	ID, StartDate, EndDate string
	Weekdays               [7]bool
}
type calendarException struct{ ServiceID, Date, Type string }
type shapePoint struct {
	Latitude, Longitude float64
	Sequence            int
}

type staticData struct {
	RouteIDs            []string
	Stops               map[string]staticStop
	StationIDs          []string
	Trips               []staticTrip
	StopTimesByTrip     map[string][]stopTime
	Calendar            []calendarService
	CalendarDates       []calendarException
	Transfers           [][2]string
	RegularRoutesByStop map[string]map[string]struct{}
	TripsByRealtimeID   map[string]staticTrip
	ConnectedStops      map[string]map[string]struct{}
	SIRLocalStopIDs     []string
	ShapesByID          map[string][]shapePoint
}

type StaticService struct {
	client   *http.Client
	realtime *RealtimeService
	mu       sync.Mutex
	data     *staticData
	loadedAt time.Time
	loading  chan struct{}
	loadErr  error
}

func NewStaticService(realtime *RealtimeService) *StaticService {
	return &StaticService{client: &http.Client{Timeout: 15 * time.Second}, realtime: realtime}
}

func (s *StaticService) Bootstrap(ctx context.Context, now time.Time) (Bootstrap, error) {
	data, err := s.load(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	var realtimeRoutes map[string]map[string]struct{}
	if s.realtime != nil {
		entities, realtimeErr := s.realtime.Entities(ctx)
		if realtimeErr == nil {
			realtimeRoutes = realtimeRoutesByStop(entities, now)
		}
	}
	return buildBootstrap(data, realtimeRoutes, now), nil
}

func (s *StaticService) load(ctx context.Context) (*staticData, error) {
	s.mu.Lock()
	if s.data != nil && time.Since(s.loadedAt) < 6*time.Hour {
		data := s.data
		s.mu.Unlock()
		return data, nil
	}
	if s.loading != nil {
		waiting := s.loading
		s.mu.Unlock()
		select {
		case <-waiting:
			s.mu.Lock()
			data, err := s.data, s.loadErr
			s.mu.Unlock()
			return data, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.loading = make(chan struct{})
	waiting := s.loading
	previous := s.data
	s.mu.Unlock()

	data, err := s.download(ctx)
	s.mu.Lock()
	if err == nil {
		s.data, s.loadedAt = data, time.Now()
	} else if previous != nil {
		data, err = previous, nil
	}
	s.loadErr = err
	close(waiting)
	s.loading = nil
	s.mu.Unlock()
	return data, err
}

func (s *StaticService) download(ctx context.Context) (*staticData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, staticURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MTA static GTFS returned %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open MTA static GTFS: %w", err)
	}
	return parseStatic(archive)
}

func parseStatic(archive *zip.Reader) (*staticData, error) {
	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	table := func(name string) ([]map[string]string, error) {
		file := files[name]
		if file == nil {
			return nil, fmt.Errorf("GTFS missing %s", name)
		}
		handle, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer handle.Close()
		rows, err := csv.NewReader(handle).ReadAll()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		if len(rows) == 0 {
			return nil, nil
		}
		headers := rows[0]
		records := make([]map[string]string, 0, len(rows)-1)
		for _, row := range rows[1:] {
			record := make(map[string]string, len(headers))
			for index, header := range headers {
				if index < len(row) {
					record[strings.TrimPrefix(header, "\ufeff")] = row[index]
				}
			}
			records = append(records, record)
		}
		return records, nil
	}
	stopsRows, err := table("stops.txt")
	if err != nil {
		return nil, err
	}
	tripRows, err := table("trips.txt")
	if err != nil {
		return nil, err
	}
	stopTimeRows, err := table("stop_times.txt")
	if err != nil {
		return nil, err
	}
	calendarRows, err := table("calendar.txt")
	if err != nil {
		return nil, err
	}
	calendarDateRows, err := table("calendar_dates.txt")
	if err != nil {
		return nil, err
	}
	transferRows, err := table("transfers.txt")
	if err != nil {
		return nil, err
	}
	shapeRows, err := table("shapes.txt")
	if err != nil {
		return nil, err
	}

	data := &staticData{
		Stops: map[string]staticStop{}, StopTimesByTrip: map[string][]stopTime{},
		TripsByRealtimeID: map[string]staticTrip{}, ConnectedStops: map[string]map[string]struct{}{},
		ShapesByID: map[string][]shapePoint{},
	}
	if _, exists := files["routes.txt"]; exists {
		rows, err := table("routes.txt")
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, row := range rows {
			id := normalizeRouteID(row["route_id"])
			if !seen[id] {
				data.RouteIDs = append(data.RouteIDs, id)
				seen[id] = true
			}
		}
	}
	for _, row := range stopsRows {
		latitude, _ := strconv.ParseFloat(row["stop_lat"], 64)
		longitude, _ := strconv.ParseFloat(row["stop_lon"], 64)
		data.Stops[row["stop_id"]] = staticStop{ID: row["stop_id"], Name: row["stop_name"], Parent: row["parent_station"], LocationType: row["location_type"], Accessible: row["wheelchair_boarding"] == "1", Latitude: latitude, Longitude: longitude}
		if row["location_type"] == "1" {
			data.StationIDs = append(data.StationIDs, row["stop_id"])
		}
	}
	for _, row := range tripRows {
		trip := staticTrip{ID: row["trip_id"], RouteID: normalizeRouteID(row["route_id"]), ServiceID: row["service_id"], DirectionID: row["direction_id"], ShapeID: row["shape_id"]}
		data.Trips = append(data.Trips, trip)
		realtimeID := trip.ID
		if separator := strings.Index(realtimeID, "_"); separator >= 0 {
			realtimeID = realtimeID[separator+1:]
		}
		data.TripsByRealtimeID[tripLookupID(realtimeID)] = trip
	}
	for _, row := range stopTimeRows {
		sequence, _ := strconv.Atoi(row["stop_sequence"])
		data.StopTimesByTrip[row["trip_id"]] = append(data.StopTimesByTrip[row["trip_id"]], stopTime{StopID: row["stop_id"], ArrivalTime: row["arrival_time"], DepartureTime: row["departure_time"], Sequence: sequence})
	}
	weekdays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	for _, row := range calendarRows {
		service := calendarService{ID: row["service_id"], StartDate: row["start_date"], EndDate: row["end_date"]}
		for index, day := range weekdays {
			service.Weekdays[index] = row[day] == "1"
		}
		data.Calendar = append(data.Calendar, service)
	}
	for _, row := range calendarDateRows {
		data.CalendarDates = append(data.CalendarDates, calendarException{ServiceID: row["service_id"], Date: row["date"], Type: row["exception_type"]})
	}
	for _, row := range transferRows {
		data.Transfers = append(data.Transfers, [2]string{row["from_stop_id"], row["to_stop_id"]})
		connectStops(data.ConnectedStops, row["from_stop_id"], row["to_stop_id"])
	}
	for _, row := range shapeRows {
		latitude, _ := strconv.ParseFloat(row["shape_pt_lat"], 64)
		longitude, _ := strconv.ParseFloat(row["shape_pt_lon"], 64)
		sequence, _ := strconv.Atoi(row["shape_pt_sequence"])
		id := row["shape_id"]
		data.ShapesByID[id] = append(data.ShapesByID[id], shapePoint{Latitude: latitude, Longitude: longitude, Sequence: sequence})
	}
	for id, points := range data.ShapesByID {
		sort.Slice(points, func(i, j int) bool { return points[i].Sequence < points[j].Sequence })
		data.ShapesByID[id] = points
	}
	for _, members := range stationComplexStopIDs {
		for index := 1; index < len(members); index++ {
			connectStops(data.ConnectedStops, members[0], members[index])
		}
	}
	for _, times := range data.StopTimesByTrip {
		sort.Slice(times, func(i, j int) bool { return times[i].Sequence < times[j].Sequence })
	}
	data.RegularRoutesByStop = regularRoutesByStop(data.Trips, data.StopTimesByTrip)
	for _, trip := range data.Trips {
		if routeFamily(trip.RouteID) != "SI" {
			continue
		}
		ids := make([]string, 0, len(data.StopTimesByTrip[trip.ID]))
		for _, item := range data.StopTimesByTrip[trip.ID] {
			ids = append(ids, stationStopID(item.StopID))
		}
		if len(ids) > len(data.SIRLocalStopIDs) {
			data.SIRLocalStopIDs = ids
		}
	}
	return data, nil
}

func buildBootstrap(data *staticData, realtime map[string]map[string]struct{}, now time.Time) Bootstrap {
	activeServices := activeServiceIDs(data, now)
	scheduled := routesByStop(data.Trips, data.StopTimesByTrip, activeServices)
	stations := make(map[string]staticStop)
	for _, id := range data.StationIDs {
		stations[id] = data.Stops[id]
	}
	sets := newDisjointSet(stations)
	resolve := func(id string) string {
		id = stationStopID(id)
		if stop, ok := data.Stops[id]; ok && stop.Parent != "" {
			return stop.Parent
		}
		if _, ok := stations[id]; ok {
			return id
		}
		return ""
	}
	for _, transfer := range data.Transfers {
		sets.join(resolve(transfer[0]), resolve(transfer[1]))
	}
	for _, members := range stationComplexStopIDs {
		for index := 1; index < len(members); index++ {
			sets.join(resolve(members[0]), resolve(members[index]))
		}
	}
	complexes := map[string][]staticStop{}
	complexOrder := []string{}
	for _, id := range data.StationIDs {
		station := stations[id]
		if len(data.RegularRoutesByStop[id]) == 0 {
			continue
		}
		root := sets.root(id)
		if len(complexes[root]) == 0 {
			complexOrder = append(complexOrder, root)
		}
		complexes[root] = append(complexes[root], station)
	}
	result := make([]Station, 0, len(complexes))
	allRoutes := map[string]struct{}{}
	for _, root := range complexOrder {
		members := complexes[root]
		nameCandidates := append([]staticStop(nil), members...)
		sort.Slice(nameCandidates, func(i, j int) bool {
			if len(nameCandidates[i].Name) == len(nameCandidates[j].Name) {
				return nameCandidates[i].Name < nameCandidates[j].Name
			}
			return len(nameCandidates[i].Name) < len(nameCandidates[j].Name)
		})
		station := Station{ID: members[0].ID, Name: nameCandidates[0].Name}
		for _, member := range members {
			station.Latitude += member.Latitude
			station.Longitude += member.Longitude
			routes := sortedRoutes(data.RegularRoutesByStop[member.ID])
			scheduledRoutes := sortedRoutes(scheduled[member.ID])
			realtimeRoutes := sortedRoutes(realtime[member.ID])
			for _, route := range routes {
				allRoutes[route] = struct{}{}
			}
			station.Services = append(station.Services, StationService{ID: member.ID, Routes: routes, ScheduledRoutes: scheduledRoutes, RealtimeRoutes: realtimeRoutes})
		}
		station.Latitude /= float64(len(members))
		station.Longitude /= float64(len(members))
		result = append(result, station)
	}
	sort.SliceStable(result, func(i, j int) bool { return naturalLess(result[i].Name, result[j].Name) })
	routes := data.RouteIDs
	if len(routes) == 0 {
		routes = sortedRoutes(allRoutes)
	}
	return Bootstrap{Routes: routes, Stations: result, GeneratedAt: now.UTC()}
}

func normalizeRouteID(id string) string {
	aliases := map[string]string{"S": "GS", "SR": "H", "SF": "FS", "SIR": "SI", "SS": "SIX"}
	if alias := aliases[id]; alias != "" {
		return alias
	}
	return id
}

func stationStopID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasSuffix(id, "N") || strings.HasSuffix(id, "S") {
		return id[:len(id)-1]
	}
	return id
}

func tripLookupID(id string) string {
	for strings.Contains(id, "..") {
		id = strings.ReplaceAll(id, "..", ".")
	}
	return id
}

func connectStops(connections map[string]map[string]struct{}, first, second string) {
	first, second = stationStopID(first), stationStopID(second)
	if first == "" || second == "" || first == second {
		return
	}
	if connections[first] == nil {
		connections[first] = map[string]struct{}{}
	}
	if connections[second] == nil {
		connections[second] = map[string]struct{}{}
	}
	connections[first][second] = struct{}{}
	connections[second][first] = struct{}{}
}

func connectedStopIDs(id string, data *staticData) map[string]struct{} {
	start := stationStopID(id)
	connected := map[string]struct{}{start: {}}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range data.ConnectedStops[current] {
			if _, found := connected[next]; found {
				continue
			}
			connected[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	return connected
}

func routeFamily(id string) string {
	id = normalizeRouteID(id)
	aliases := map[string]string{"6X": "6", "7X": "7", "FX": "F", "SIX": "SI"}
	if family := aliases[id]; family != "" {
		return family
	}
	return id
}

func regularRoutesByStop(trips []staticTrip, times map[string][]stopTime) map[string]map[string]struct{} {
	tripCounts := map[string]int{}
	stopCounts := map[string]map[string]int{}
	for _, trip := range trips {
		route := normalizeRouteID(trip.RouteID)
		if route == "" {
			continue
		}
		tripCounts[route]++
		visited := map[string]struct{}{}
		for _, item := range times[trip.ID] {
			visited[stationStopID(item.StopID)] = struct{}{}
		}
		for stop := range visited {
			if stopCounts[stop] == nil {
				stopCounts[stop] = map[string]int{}
			}
			stopCounts[stop][route]++
		}
	}
	result := map[string]map[string]struct{}{}
	for stop, counts := range stopCounts {
		for route, count := range counts {
			if float64(count)/float64(tripCounts[route]) >= .05 {
				if result[stop] == nil {
					result[stop] = map[string]struct{}{}
				}
				result[stop][route] = struct{}{}
			}
		}
	}
	return result
}

func routesByStop(trips []staticTrip, times map[string][]stopTime, active map[string]struct{}) map[string]map[string]struct{} {
	result := map[string]map[string]struct{}{}
	for _, trip := range trips {
		if _, ok := active[trip.ServiceID]; !ok {
			continue
		}
		visited := map[string]struct{}{}
		for _, item := range times[trip.ID] {
			visited[stationStopID(item.StopID)] = struct{}{}
		}
		for stop := range visited {
			if result[stop] == nil {
				result[stop] = map[string]struct{}{}
			}
			result[stop][normalizeRouteID(trip.RouteID)] = struct{}{}
		}
	}
	return result
}

func activeServiceIDs(data *staticData, now time.Time) map[string]struct{} {
	now = now.In(newYorkLocation)
	date := now.Format("20060102")
	weekday := (int(now.Weekday()) + 6) % 7
	active := map[string]struct{}{}
	for _, service := range data.Calendar {
		if service.StartDate <= date && service.EndDate >= date && service.Weekdays[weekday] {
			active[service.ID] = struct{}{}
		}
	}
	for _, exception := range data.CalendarDates {
		if exception.Date != date {
			continue
		}
		if exception.Type == "1" {
			active[exception.ServiceID] = struct{}{}
		}
		if exception.Type == "2" {
			delete(active, exception.ServiceID)
		}
	}
	return active
}

func sortedRoutes(values map[string]struct{}) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for value := range values {
		value = normalizeRouteID(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return naturalLess(result[i], result[j]) })
	return result
}

func naturalLess(left, right string) bool {
	left, right = strings.ToLower(left), strings.ToLower(right)
	first, second := 0, 0
	for first < len(left) && second < len(right) {
		if left[first] >= '0' && left[first] <= '9' && right[second] >= '0' && right[second] <= '9' {
			firstEnd, secondEnd := first, second
			for firstEnd < len(left) && left[firstEnd] >= '0' && left[firstEnd] <= '9' {
				firstEnd++
			}
			for secondEnd < len(right) && right[secondEnd] >= '0' && right[secondEnd] <= '9' {
				secondEnd++
			}
			a, b := strings.TrimLeft(left[first:firstEnd], "0"), strings.TrimLeft(right[second:secondEnd], "0")
			if len(a) != len(b) {
				return len(a) < len(b)
			}
			if a != b {
				return a < b
			}
			first, second = firstEnd, secondEnd
			continue
		}
		if left[first] != right[second] {
			return left[first] < right[second]
		}
		first++
		second++
	}
	return first == len(left) && second < len(right)
}

type disjointSet map[string]string

func newDisjointSet(values map[string]staticStop) disjointSet {
	set := disjointSet{}
	for id := range values {
		set[id] = id
	}
	return set
}
func (s disjointSet) root(id string) string {
	if id == "" {
		return ""
	}
	parent, ok := s[id]
	if !ok {
		return ""
	}
	if parent != id {
		s[id] = s.root(parent)
	}
	return s[id]
}
func (s disjointSet) join(left, right string) {
	left, right = s.root(left), s.root(right)
	if left != "" && right != "" && left != right {
		s[right] = left
	}
}
