package mta

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

var routeColors = map[string]string{
	"1": "EE352E", "2": "EE352E", "3": "EE352E", "4": "00933C", "5": "00933C", "6": "00933C", "6X": "00933C",
	"7": "B933AD", "7X": "B933AD", "A": "0039A6", "C": "0039A6", "E": "0039A6", "B": "FF6319", "D": "FF6319",
	"F": "FF6319", "FX": "FF6319", "M": "FF6319", "G": "6CBE45", "J": "996633", "Z": "996633", "L": "A7A9AC",
	"N": "FCCC0A", "Q": "FCCC0A", "R": "FCCC0A", "W": "FCCC0A", "GS": "808183", "FS": "808183", "H": "808183",
	"SI": "0039A6", "SIX": "0039A6",
}

func displayTripRouteID(routeID string, trip staticTrip, realtime []*gtfs.TripUpdate_StopTimeUpdate, data *staticData) string {
	routeID = normalizeRouteID(routeID)
	if routeID != "SI" {
		return routeID
	}
	stops := []string{}
	if trip.ID != "" {
		for _, item := range data.StopTimesByTrip[trip.ID] {
			stops = append(stops, item.StopID)
		}
	} else {
		for _, item := range realtime {
			stops = append(stops, item.GetStopId())
		}
	}
	previous := -1
	for _, stop := range stops {
		index := -1
		for position, id := range data.SIRLocalStopIDs {
			if id == stationStopID(stop) {
				index = position
				break
			}
		}
		if index < 0 {
			continue
		}
		if previous >= 0 && abs(index-previous) > 1 {
			return "SIX"
		}
		previous = index
	}
	return "SI"
}

type StopObject struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Accessible bool     `json:"accessible"`
	Borough    string   `json:"borough,omitempty"`
}
type StopEvent struct {
	Time int64 `json:"time"
	_ "time/tzdata"`
}
type ArrivalTrip struct {
	ID    string `json:"id"`
	Route struct {
		ID    string `json:"id"`
		Color string `json:"color"`
	} `json:"route"`
}
type Arrival struct {
	Stop           StopObject  `json:"stop"`
	Arrival        *StopEvent  `json:"arrival,omitempty"`
	Departure      *StopEvent  `json:"departure,omitempty"`
	Future         bool        `json:"future"`
	NotDeparted    bool        `json:"notDeparted"`
	DelayMinutes   *int        `json:"delayMinutes,omitempty"`
	Trip           ArrivalTrip `json:"trip"`
	Destination    StopObject  `json:"destination"`
	Headsign       string      `json:"headsign"`
	TransferRoutes []string    `json:"transferRoutes,omitempty"`
	Estimated      bool        `json:"estimated,omitempty"`
}
type Transfer struct {
	Stop   StopObject `json:"stop"`
	Routes []string   `json:"routes"`
}
type StopResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Borough     string           `json:"borough,omitempty"`
	Accessible  bool             `json:"accessible"`
	StopTimes   []Arrival        `json:"stopTimes"`
	Services    []StationService `json:"services"`
	Transfers   []Transfer       `json:"transfers"`
	GeneratedAt int64            `json:"generatedAt"`
}

type TransitService struct {
	static   *StaticService
	realtime *RealtimeService
	metadata *MetadataService
}

func NewTransitService(static *StaticService, realtime *RealtimeService, metadata *MetadataService) *TransitService {
	return &TransitService{static: static, realtime: realtime, metadata: metadata}
}

func (s *TransitService) Stop(ctx context.Context, id string, now time.Time) (StopResponse, error) {
	data, err := s.static.load(ctx)
	if err != nil {
		return StopResponse{}, err
	}
	entities, err := s.realtime.Entities(ctx)
	if err != nil {
		return StopResponse{}, err
	}
	metadata := s.metadata.Get(ctx)
	nowUnix := now.Unix()
	serviceStart := nowUnix - int64(currentNewYorkSecond(now))
	routesByStop := realtimeRoutesByStop(entities, now)
	baseID := stationStopID(id)
	baseName := strings.ToLower(strings.TrimSpace(stopObject(baseID, data, metadata).Name))
	connected := connectedStopIDs(baseID, data)
	stationIDs := map[string]struct{}{baseID: {}}
	for connectedID := range connected {
		if strings.ToLower(strings.TrimSpace(stopObject(connectedID, data, metadata).Name)) == baseName {
			stationIDs[connectedID] = struct{}{}
		}
	}
	services := make([]StationService, 0, len(stationIDs))
	for stationID := range stationIDs {
		families := map[string]struct{}{}
		for route := range data.RegularRoutesByStop[stationID] {
			families[routeFamily(route)] = struct{}{}
		}
		routes := sortedRoutes(families)
		if len(routes) > 0 {
			services = append(services, StationService{ID: stationID, Routes: routes})
		}
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].ID == baseID || services[j].ID == baseID {
			return services[i].ID == baseID
		}
		return services[i].ID < services[j].ID
	})

	arrivals := make([]Arrival, 0)
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		if update.GetTrip() == nil {
			continue
		}
		var match *gtfs.TripUpdate_StopTimeUpdate
		for _, item := range update.GetStopTimeUpdate() {
			if _, found := stationIDs[stationStopID(item.GetStopId())]; found {
				match = item
				break
			}
		}
		if match == nil {
			continue
		}
		timestamp := eventTimestamp(match)
		if timestamp == 0 || timestamp < nowUnix {
			continue
		}
		destinationID := ""
		for index := len(update.GetStopTimeUpdate()) - 1; index >= 0; index-- {
			candidate := update.GetStopTimeUpdate()[index].GetStopId()
			if isPublicStop(candidate, data) {
				destinationID = candidate
				break
			}
		}
		staticTrip := data.TripsByRealtimeID[tripLookupID(update.GetTrip().GetTripId())]
		routeID := normalizeRouteID(update.GetTrip().GetRouteId())
		staticTimes := data.StopTimesByTrip[staticTrip.ID]
		tripStartSecond := gtfsSecond(update.GetTrip().GetStartTime())
		if tripStartSecond < 0 {
			tripStartSecond = tripIDStartSecond(update.GetTrip().GetTripId())
		}
		if tripStartSecond < 0 && len(staticTimes) > 0 {
			tripStartSecond = gtfsSecond(firstTime(staticTimes[0]))
		}
		tripStart := tripStartTimestamp(update.GetTrip().GetStartDate(), tripStartSecond)
		if tripStart == 0 && tripStartSecond >= 0 {
			tripStart = serviceStart + int64(tripStartSecond)
		}
		reportedDelay := int64(update.GetDelay())
		if arrival := match.GetArrival(); arrival != nil {
			reportedDelay = int64(arrival.GetDelay())
		} else if departure := match.GetDeparture(); departure != nil {
			reportedDelay = int64(departure.GetDelay())
		}
		var delayMinutes *int
		if reportedDelay >= 180 && reportedDelay < 7200 {
			delay := int(math.Round(float64(reportedDelay) / 60))
			delayMinutes = &delay
		}
		arrival := Arrival{Stop: staticStopObject(match.GetStopId(), data), Future: true, NotDeparted: tripStart > nowUnix, DelayMinutes: delayMinutes, Destination: staticStopObject(destinationID, data)}
		if match.GetArrival().GetTime() != 0 {
			arrival.Arrival = &StopEvent{Time: match.GetArrival().GetTime()}
		}
		if match.GetDeparture().GetTime() != 0 {
			arrival.Departure = &StopEvent{Time: match.GetDeparture().GetTime()}
		}
		arrival.Trip.ID = update.GetTrip().GetTripId()
		arrival.Trip.Route.ID = displayTripRouteID(routeID, staticTrip, update.GetStopTimeUpdate(), data)
		arrival.Trip.Route.Color = routeColors[routeID]
		arrival.Destination.Borough = metadata.BoroughByStopID[stationStopID(arrival.Destination.ID)]
		arrival.Headsign = arrival.Destination.Name
		arrivals = append(arrivals, arrival)
	}
	sort.SliceStable(arrivals, func(i, j int) bool { return arrivalTimestamp(arrivals[i]) < arrivalTimestamp(arrivals[j]) })

	transfers := []Transfer{}
	baseRoutes := data.RegularRoutesByStop[baseID]
	for transferID := range connected {
		if _, same := stationIDs[transferID]; same {
			continue
		}
		routes := []string{}
		for route := range routesByStop[transferID] {
			family := routeFamily(route)
			overlaps := false
			for baseRoute := range baseRoutes {
				if routeFamily(baseRoute) == family {
					overlaps = true
					break
				}
			}
			if !overlaps {
				routes = append(routes, family)
			}
		}
		routes = uniqueStrings(routes)
		sort.Slice(routes, func(i, j int) bool { return naturalLess(routes[i], routes[j]) })
		if len(routes) > 0 {
			transfers = append(transfers, Transfer{Stop: staticStopObject(transferID, data), Routes: routes})
		}
	}
	base := stopObject(id, data, metadata)
	return StopResponse{ID: id, Name: base.Name, Borough: base.Borough, Accessible: base.Accessible, StopTimes: arrivals, Services: services, Transfers: transfers, GeneratedAt: nowUnix}, nil
}

func staticStopObject(id string, data *staticData) StopObject {
	stop, found := data.Stops[id]
	if !found {
		stop = data.Stops[stationStopID(id)]
	}
	result := StopObject{ID: id, Name: stop.Name, Accessible: stop.Accessible}
	if stop.Latitude != 0 || stop.Longitude != 0 {
		lat, lon := stop.Latitude, stop.Longitude
		result.Latitude, result.Longitude = &lat, &lon
	}
	if result.Name == "" {
		result.Name = id
	}
	return result
}

func stopObject(id string, data *staticData, metadata StationMetadata) StopObject {
	result := staticStopObject(id, data)
	result.Borough = metadata.BoroughByStopID[stationStopID(id)]
	_, result.Accessible = metadata.AccessibleIDs[stationStopID(id)]
	return result
}

func isPublicStop(id string, data *staticData) bool {
	_, exact := data.Stops[id]
	_, base := data.Stops[stationStopID(id)]
	return id != "" && (exact || base)
}
func eventTimestamp(item *gtfs.TripUpdate_StopTimeUpdate) int64 {
	if item.GetArrival().GetTime() != 0 {
		return item.GetArrival().GetTime()
	}
	return item.GetDeparture().GetTime()
}
func arrivalTimestamp(item Arrival) int64 {
	if item.Arrival != nil {
		return item.Arrival.Time
	}
	if item.Departure != nil {
		return item.Departure.Time
	}
	return math.MaxInt64
}
func firstTime(item stopTime) string { return firstNonempty(item.DepartureTime, item.ArrivalTime) }
func lastTime(item stopTime) string  { return firstNonempty(item.ArrivalTime, item.DepartureTime) }
func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func gtfsSecond(value string) int {
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return -1
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil {
		return -1
	}
	s := 0
	if len(parts) > 2 {
		s, _ = strconv.Atoi(parts[2])
	}
	return h*3600 + m*60 + s
}
func tripIDStartSecond(id string) int {
	part := strings.Split(id, "_")[0]
	if part == "" {
		return -1
	}
	value, err := strconv.Atoi(part)
	if err != nil {
		return -1
	}
	return int(math.Round(float64(value) * .6))
}
func tripStartTimestamp(date string, second int) int64 {
	if len(date) != 8 || second < 0 {
		return 0
	}
	parsed, err := time.ParseInLocation("20060102", date, newYorkLocation)
	if err != nil {
		return 0
	}
	noon := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, newYorkLocation)
	return noon.Unix() - 12*3600 + int64(second)
}
func currentNewYorkSecond(now time.Time) int {
	local := now.In(newYorkLocation)
	return local.Hour()*3600 + local.Minute()*60 + local.Second()
}

var newYorkLocation = func() *time.Location {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return location
}()

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := values[:0]
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

func tripHasDeparted(update *gtfs.TripUpdate, data *staticData, now time.Time) bool {
	if update.GetTrip() == nil || update.GetTrip().GetScheduleRelationship() == gtfs.TripDescriptor_CANCELED {
		return false
	}
	trip := data.TripsByRealtimeID[tripLookupID(update.GetTrip().GetTripId())]
	stops := data.StopTimesByTrip[trip.ID]
	second := gtfsSecond(update.GetTrip().GetStartTime())
	if second < 0 {
		second = tripIDStartSecond(update.GetTrip().GetTripId())
	}
	if second < 0 && len(stops) > 0 {
		second = gtfsSecond(firstTime(stops[0]))
	}
	start := tripStartTimestamp(update.GetTrip().GetStartDate(), second)
	if start == 0 && second >= 0 {
		start = now.Unix() - int64(currentNewYorkSecond(now)) + int64(second)
	}
	if start > now.Unix() {
		return false
	}
	if len(stops) > 0 {
		for _, item := range update.GetStopTimeUpdate() {
			if item.GetStopId() != stops[0].StopID {
				continue
			}
			departure := item.GetDeparture().GetTime()
			if departure == 0 {
				departure = item.GetArrival().GetTime()
			}
			if departure > now.Unix() {
				return false
			}
		}
	}
	return true
}
