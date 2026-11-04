package mta

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type TripResponse struct {
	ID        string     `json:"id"`
	UpdatedAt int64      `json:"updatedAt"`
	Origin    StopObject `json:"origin"`
	Route     struct {
		ID    string `json:"id"`
		Color string `json:"color"`
	} `json:"route"`
	MapCoordinates [][]float64 `json:"mapCoordinates"`
	StopTimes      []Arrival   `json:"stopTimes"`
	Vehicle        *MapVehicle `json:"vehicle"`
}

func (s *TransitService) Trip(ctx context.Context, requestedRoute, tripID string, now time.Time) (TripResponse, error) {
	data, err := s.static.load(ctx)
	if err != nil {
		return TripResponse{}, err
	}
	entities, err := s.realtime.Entities(ctx)
	if err != nil {
		return TripResponse{}, err
	}
	metadata := s.metadata.Get(ctx)
	var update *gtfs.TripUpdate
	for _, entity := range entities {
		candidate := entity.GetTripUpdate()
		if candidate.GetTrip().GetTripId() == tripID {
			update = candidate
			break
		}
	}
	if update == nil {
		return TripResponse{}, errors.New("MTA realtime trip not found")
	}
	routeID := normalizeRouteID(update.GetTrip().GetRouteId())
	if routeID == "" {
		routeID = normalizeRouteID(requestedRoute)
	}
	staticTrip := findStaticTrip(data, update, tripID, routeID, activeServiceIDs(data, now))
	hasStatic := staticTrip.ID != ""
	staticTimes := data.StopTimesByTrip[staticTrip.ID]
	realtimeTimes := []*gtfs.TripUpdate_StopTimeUpdate{}
	for _, item := range update.GetStopTimeUpdate() {
		if isPublicStop(item.GetStopId(), data) {
			realtimeTimes = append(realtimeTimes, item)
		}
	}
	displayedStatic := trimStaticToTerminal(realtimeTimes, staticTimes)
	authoritative := realtimePathDiffers(realtimeTimes, displayedStatic)
	stopTimes := []Arrival{}
	if hasStatic && len(displayedStatic) > 0 && !authoritative {
		exact := staticTrip.ID == tripID || strings.HasSuffix(staticTrip.ID, "_"+tripID)
		stopTimes = mergeStaticRealtime(displayedStatic, realtimeTimes, update, data, metadata, now, exact)
	} else {
		for _, item := range realtimeTimes {
			stopTimes = append(stopTimes, arrivalFromRealtime(item, update, data, metadata, now))
		}
	}
	routesByStop := realtimeRoutesByStop(entities, now)
	for index := range stopTimes {
		stopTimes[index].TransferRoutes = interchangeRoutes(stopTimes[index].Stop.ID, data, routeID, routesByStop)
	}
	coordinates := [][]float64{}
	if !authoritative && hasStatic && staticTrip.ShapeID != "" {
		points := data.ShapesByID[staticTrip.ShapeID]
		stride := 1
		if len(points) > 700 {
			stride = (len(points) + 699) / 700
		}
		for index, point := range points {
			if index%stride == 0 || index == len(points)-1 {
				coordinates = append(coordinates, []float64{point.Longitude, point.Latitude})
			}
		}
	}
	if len(coordinates) < 2 {
		coordinates = nil
		for _, item := range stopTimes {
			if item.Stop.Longitude != nil && item.Stop.Latitude != nil {
				coordinates = append(coordinates, []float64{*item.Stop.Longitude, *item.Stop.Latitude})
			}
		}
	}
	originID := ""
	if len(staticTimes) > 0 {
		originID = staticTimes[0].StopID
	} else if len(realtimeTimes) > 0 {
		originID = realtimeTimes[0].GetStopId()
	}
	result := TripResponse{ID: tripID, UpdatedAt: now.Unix(), Origin: staticStopObject(originID, data), MapCoordinates: coordinates, StopTimes: stopTimes}
	result.Route.ID = displayTripRouteID(firstNonempty(staticTrip.RouteID, routeID), staticTrip, update.GetStopTimeUpdate(), data)
	result.Route.Color = routeColors[routeID]
	line := MapFeature{Type: "Feature"}
	line.Properties.Color = "#" + routeColors[routeFamily(routeID)]
	line.Geometry.Coordinates = coordinates
	tripEntities := []*gtfs.FeedEntity{}
	for _, entity := range entities {
		if entity.GetTripUpdate().GetTrip().GetTripId() == tripID || entity.GetVehicle().GetTrip().GetTripId() == tripID {
			tripEntities = append(tripEntities, entity)
		}
	}
	vehicles := mapVehicles(tripEntities, data, map[string]struct{}{routeFamily(routeID): {}}, []MapFeature{line}, now)
	if len(vehicles) > 0 {
		for _, entity := range tripEntities {
			if entity.GetVehicle().GetPosition() != nil {
				vehicles[0].RouteID = result.Route.ID
				break
			}
		}
		result.Vehicle = &vehicles[0]
	}
	return result, nil
}

func arrivalFromRealtime(item *gtfs.TripUpdate_StopTimeUpdate, update *gtfs.TripUpdate, data *staticData, metadata StationMetadata, now time.Time) Arrival {
	timestamp := eventTimestamp(item)
	destinationID := ""
	for index := len(update.GetStopTimeUpdate()) - 1; index >= 0; index-- {
		candidate := update.GetStopTimeUpdate()[index].GetStopId()
		if isPublicStop(candidate, data) {
			destinationID = candidate
			break
		}
	}
	arrival := Arrival{Stop: staticStopObject(item.GetStopId(), data), Future: timestamp >= now.Unix(), Destination: staticStopObject(destinationID, data)}
	if item.GetArrival().GetTime() != 0 {
		arrival.Arrival = &StopEvent{Time: item.GetArrival().GetTime()}
	}
	if item.GetDeparture().GetTime() != 0 {
		arrival.Departure = &StopEvent{Time: item.GetDeparture().GetTime()}
	}
	arrival.Headsign = arrival.Destination.Name
	return arrival
}

func mergeStaticRealtime(staticTimes []stopTime, realtime []*gtfs.TripUpdate_StopTimeUpdate, update *gtfs.TripUpdate, data *staticData, metadata StationMetadata, now time.Time, exact bool) []Arrival {
	byStop := map[string]*gtfs.TripUpdate_StopTimeUpdate{}
	bySequence := map[int]*gtfs.TripUpdate_StopTimeUpdate{}
	for _, item := range realtime {
		byStop[item.GetStopId()] = item
		bySequence[int(item.GetStopSequence())] = item
	}
	type merged struct {
		index, scheduled int
		timestamp        int64
		arrival          Arrival
	}
	values := make([]merged, 0, len(staticTimes))
	for index, item := range staticTimes {
		match := byStop[item.StopID]
		if match == nil && exact {
			match = bySequence[item.Sequence]
		}
		scheduled := gtfsSecond(firstNonempty(item.ArrivalTime, item.DepartureTime))
		arrival := Arrival{Stop: staticStopObject(item.StopID, data)}
		timestamp := int64(0)
		if match != nil {
			arrival = arrivalFromRealtime(match, update, data, metadata, now)
			timestamp = eventTimestamp(match)
		}
		values = append(values, merged{index: index, scheduled: scheduled, timestamp: timestamp, arrival: arrival})
	}
	anchors := []merged{}
	for _, value := range values {
		if value.timestamp > 0 && value.scheduled >= 0 {
			anchors = append(anchors, value)
		}
	}
	result := make([]Arrival, 0, len(values))
	for _, value := range values {
		arrival := value.arrival
		if value.timestamp == 0 && value.scheduled >= 0 && len(anchors) > 0 {
			nearest := anchors[0]
			for _, candidate := range anchors[1:] {
				if abs(candidate.index-value.index) < abs(nearest.index-value.index) {
					nearest = candidate
				}
			}
			estimated := int64(value.scheduled) + nearest.timestamp - int64(nearest.scheduled)
			arrival.Arrival = &StopEvent{Time: estimated}
			arrival.Departure = &StopEvent{Time: estimated}
			arrival.Future = estimated >= now.Unix()
			arrival.Estimated = true
		}
		result = append(result, arrival)
	}
	return result
}

func realtimePathDiffers(realtime []*gtfs.TripUpdate_StopTimeUpdate, static []stopTime) bool {
	if len(realtime) == 0 || len(static) == 0 {
		return false
	}
	bySequence := map[int]string{}
	for _, item := range static {
		bySequence[item.Sequence] = item.StopID
	}
	for _, item := range realtime {
		if item.StopSequence != nil {
			if staticID := bySequence[int(item.GetStopSequence())]; staticID != "" && staticID != item.GetStopId() {
				return true
			}
		}
	}
	staticIndex := -1
	for _, item := range realtime {
		next := -1
		for index := staticIndex + 1; index < len(static); index++ {
			if static[index].StopID == item.GetStopId() {
				next = index
				break
			}
		}
		if next < 0 {
			return true
		}
		staticIndex = next
	}
	return false
}
func trimStaticToTerminal(realtime []*gtfs.TripUpdate_StopTimeUpdate, static []stopTime) []stopTime {
	terminal := ""
	for index := len(realtime) - 1; index >= 0; index-- {
		if realtime[index].GetStopId() != "" {
			terminal = realtime[index].GetStopId()
			break
		}
	}
	if terminal == "" {
		return static
	}
	terminalIndex := -1
	for index := len(static) - 1; index >= 0; index-- {
		if static[index].StopID == terminal {
			terminalIndex = index
			break
		}
	}
	if terminalIndex < 0 || terminalIndex == len(static)-1 {
		return static
	}
	return static[:terminalIndex+1]
}
func findStaticTrip(data *staticData, update *gtfs.TripUpdate, tripID, routeID string, active map[string]struct{}) staticTrip {
	matches := []staticTrip{}
	for _, trip := range data.Trips {
		if trip.ID == tripID || strings.HasSuffix(trip.ID, "_"+tripID) {
			matches = append(matches, trip)
		}
	}
	encoded := strings.Split(tripID, "_")[0]
	encodedValue, encodedErr := strconv.Atoi(encoded)
	startMinute := float64(encodedValue) / 100
	direction := ""
	if len(update.GetStopTimeUpdate()) > 0 {
		id := update.GetStopTimeUpdate()[0].GetStopId()
		if strings.HasSuffix(id, "N") || strings.HasSuffix(id, "S") {
			direction = id[len(id)-1:]
		}
	}
	if len(matches) == 0 && encodedErr == nil {
		for _, trip := range data.Trips {
			if routeFamily(trip.RouteID) != routeFamily(routeID) {
				continue
			}
			if _, ok := active[trip.ServiceID]; !ok {
				continue
			}
			times := data.StopTimesByTrip[trip.ID]
			if len(times) == 0 {
				continue
			}
			firstMinute := float64(gtfsSecond(firstTime(times[0]))) / 60
			matchesDirection := direction == "" || strings.HasSuffix(times[0].StopID, direction)
			if matchesDirection && absFloat(firstMinute-startMinute) <= 10 {
				matches = append(matches, trip)
			}
		}
	}
	realtimeStops := map[string]struct{}{}
	for _, item := range update.GetStopTimeUpdate() {
		realtimeStops[item.GetStopId()] = struct{}{}
	}
	type scored struct {
		trip  staticTrip
		score float64
	}
	scores := []scored{}
	for _, trip := range matches {
		times := data.StopTimesByTrip[trip.ID]
		matching := 0
		for _, item := range times {
			if _, ok := realtimeStops[item.StopID]; ok {
				matching++
			}
		}
		firstMinute := 0.0
		if len(times) > 0 {
			firstMinute = float64(gtfsSecond(firstTime(times[0]))) / 60
		}
		difference := 0.0
		if encodedErr == nil {
			difference = absFloat(firstMinute - startMinute)
		}
		score := float64(matching*1000+len(times)) - difference*10
		if _, ok := active[trip.ServiceID]; ok {
			score += 100
		}
		scores = append(scores, scored{trip, score})
	}
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if len(scores) > 0 {
		return scores[0].trip
	}
	return staticTrip{}
}
func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
