package mta

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

type MapFeature struct {
	Type       string `json:"type"`
	Properties struct {
		RouteID string  `json:"routeId"`
		Color   string  `json:"color"`
		Offset  float64 `json:"offset"`
	} `json:"properties"`
	Geometry struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"geometry"`
}

type MapResponse struct {
	Lines    []MapFeature `json:"lines"`
	Stations []Station    `json:"stations"`
	Vehicles []MapVehicle `json:"vehicles"`
}

type MapVehicle struct {
	Bearing          float64 `json:"bearing"`
	Offset           float64 `json:"offset"`
	ID               string  `json:"id"`
	RouteID          string  `json:"routeId"`
	Longitude        float64 `json:"longitude"`
	Latitude         float64 `json:"latitude"`
	DirectionBearing float64 `json:"directionBearing"`
}

type StationMapResponse struct {
	Lines    []MapFeature `json:"lines"`
	Station  Station      `json:"station"`
	Vehicles []MapVehicle `json:"vehicles"`
}

var mapOffsets = map[string]float64{"0039A6": -8, "FCCC0A": 0, "FF6319": 8, "6CBE45": -16, "996633": 16, "A7A9AC": 24, "EE352E": -24, "00933C": 32, "B933AD": -32}

func (s *StaticService) Map(ctx context.Context, now time.Time) (MapResponse, error) {
	data, err := s.load(ctx)
	if err != nil {
		return MapResponse{}, err
	}
	entities, err := s.realtime.Entities(ctx)
	if err != nil {
		return MapResponse{}, err
	}
	bootstrap := buildBootstrap(data, realtimeRoutesByStop(entities, now), now)
	lines := currentRouteLines(data, entities, now)
	displayed := map[string]struct{}{}
	for _, station := range bootstrap.Stations {
		for _, route := range mapStationRoutes(station) {
			displayed[route] = struct{}{}
		}
	}
	return MapResponse{Lines: lines, Stations: bootstrap.Stations, Vehicles: mapVehicles(entities, data, displayed, lines, now)}, nil
}

func mapStationRoutes(station Station) []string {
	realtime, scheduled, regular := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, service := range station.Services {
		for _, route := range service.RealtimeRoutes {
			realtime[routeFamily(route)] = struct{}{}
		}
		for _, route := range service.ScheduledRoutes {
			scheduled[routeFamily(route)] = struct{}{}
		}
		for _, route := range service.Routes {
			regular[routeFamily(route)] = struct{}{}
		}
	}
	if len(realtime) > 0 {
		return sortedRoutes(realtime)
	}
	if len(scheduled) > 0 {
		return sortedRoutes(scheduled)
	}
	return sortedRoutes(regular)
}

func currentRouteLines(data *staticData, entities []*gtfs.FeedEntity, now time.Time) []MapFeature {
	activeRoutes := map[string]bool{}
	matchedRoutes := map[string]bool{}
	matchedTrips := []staticTrip{}
	seen := map[string]bool{}
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		upcoming := false
		for _, stop := range update.GetStopTimeUpdate() {
			if mapStopTimestamp(stop) >= now.Unix() {
				upcoming = true
				break
			}
		}
		if !upcoming {
			continue
		}
		route := routeFamily(update.GetTrip().GetRouteId())
		activeRoutes[route] = true
		trip := data.TripsByRealtimeID[tripLookupID(update.GetTrip().GetTripId())]
		if trip.ShapeID == "" {
			continue
		}
		matchedRoutes[route] = true
		if !seen[trip.ID] {
			trip.RouteID = route
			matchedTrips = append(matchedTrips, trip)
			seen[trip.ID] = true
		}
	}
	fallback := []staticTrip{}
	for _, trip := range data.Trips {
		route := routeFamily(trip.RouteID)
		if activeRoutes[route] && !matchedRoutes[route] {
			fallback = append(fallback, trip)
		}
	}
	lines := routeLineFeatures(matchedTrips, data.ShapesByID, nil)
	if len(fallback) > 0 {
		lines = append(lines, routeLineFeatures(fallback, data.ShapesByID, activeServiceIDs(data, now))...)
	}
	return lines
}

func routeLineFeatures(trips []staticTrip, shapes map[string][]shapePoint, services map[string]struct{}) []MapFeature {
	pool := trips
	if services != nil {
		eligible := []staticTrip{}
		for _, trip := range trips {
			if _, ok := services[trip.ServiceID]; ok {
				eligible = append(eligible, trip)
			}
		}
		if len(eligible) > 0 {
			pool = eligible
		}
	}
	counts := map[string]map[string]int{}
	routes := []string{}
	for _, trip := range pool {
		route := routeFamily(trip.RouteID)
		if route == "" || trip.ShapeID == "" || len(shapes[trip.ShapeID]) == 0 {
			continue
		}
		if counts[route] == nil {
			counts[route] = map[string]int{}
			routes = append(routes, route)
		}
		counts[route][trip.ShapeID]++
	}
	features := []MapFeature{}
	for _, route := range routes {
		ids := make([]string, 0, len(counts[route]))
		for id := range counts[route] {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			if counts[route][ids[i]] == counts[route][ids[j]] {
				return ids[i] < ids[j]
			}
			return counts[route][ids[i]] > counts[route][ids[j]]
		})
		terminals := map[string]bool{}
		selected := 0
		for _, id := range ids {
			points := shapes[id]
			ends := []string{
				fmt.Sprintf("%.4f,%.4f", points[0].Longitude, points[0].Latitude),
				fmt.Sprintf("%.4f,%.4f", points[len(points)-1].Longitude, points[len(points)-1].Latitude),
			}
			sort.Strings(ends)
			key := ends[0] + "|" + ends[1]
			if terminals[key] {
				continue
			}
			terminals[key] = true
			selected++
			if selected > 6 {
				break
			}
			coordinates := shapeCoordinates(points)
			if len(coordinates) < 2 {
				continue
			}
			feature := MapFeature{Type: "Feature"}
			feature.Properties.RouteID = route
			color := routeColors[route]
			if color == "" {
				color = "808183"
			}
			feature.Properties.Color = "#" + color
			feature.Properties.Offset = mapOffsets[color]
			feature.Geometry.Type = "LineString"
			feature.Geometry.Coordinates = coordinates
			features = append(features, feature)
		}
	}
	return mergeMapCorridors(features)
}

func mapStopTimestamp(item *gtfs.TripUpdate_StopTimeUpdate) int64 {
	if arrival := item.GetArrival(); arrival != nil {
		return arrival.GetTime()
	}
	return item.GetDeparture().GetTime()
}

func mapCoordinateDistance(a, b []float64) float64 {
	return math.Hypot((a[0]-b[0])*math.Cos(a[1]*math.Pi/180), a[1]-b[1]) * 111320
}

func mergeMapCorridors(features []MapFeature) []MapFeature {
	result := []MapFeature{}
	for _, feature := range features {
		points := append([][]float64(nil), feature.Geometry.Coordinates...)
		if len(points) < 2 {
			continue
		}
		first, last := points[0], points[len(points)-1]
		if first[0] > last[0] || first[0] == last[0] && first[1] > last[1] {
			for left, right := 0, len(points)-1; left < right; left, right = left+1, right-1 {
				points[left], points[right] = points[right], points[left]
			}
		}
		duplicate := false
		for _, existing := range result {
			path := existing.Geometry.Coordinates
			if existing.Properties.RouteID != feature.Properties.RouteID || existing.Properties.Color != feature.Properties.Color || mapCoordinateDistance(path[0], points[0]) > 200 || mapCoordinateDistance(path[len(path)-1], points[len(points)-1]) > 200 {
				continue
			}
			covered := true
			for _, point := range points {
				near := false
				for index := 1; index < len(path); index++ {
					start, end := path[index-1], path[index]
					scale := math.Cos(point[1] * math.Pi / 180)
					dx, dy := (end[0]-start[0])*scale, end[1]-start[1]
					fraction := 0.0
					if length := dx*dx + dy*dy; length > 0 {
						fraction = math.Max(0, math.Min(1, ((point[0]-start[0])*scale*dx+(point[1]-start[1])*dy)/length))
					}
					if mapCoordinateDistance(point, []float64{start[0] + (end[0]-start[0])*fraction, start[1] + dy*fraction}) <= 60 {
						near = true
						break
					}
				}
				if !near {
					covered = false
					break
				}
			}
			if covered {
				duplicate = true
				break
			}
		}
		if !duplicate {
			feature.Geometry.Coordinates = points
			result = append(result, feature)
		}
	}
	return result
}

func shapeCoordinates(points []shapePoint) [][]float64 {
	stride := 1
	if len(points) > 500 {
		stride = (len(points) + 499) / 500
	}
	coordinates := [][]float64{}
	for index, point := range points {
		if index%stride == 0 || index == len(points)-1 {
			coordinates = append(coordinates, []float64{point.Longitude, point.Latitude})
		}
	}
	return coordinates
}

func coordinateBearing(start, end []float64) float64 {
	latitude := (start[1] + end[1]) / 2
	longitudeScale := math.Cos(latitude * math.Pi / 180)
	return math.Atan2((end[0]-start[0])*longitudeScale, end[1]-start[1]) * 180 / math.Pi
}

func stopCoordinate(id string, data *staticData) []float64 {
	stop, ok := data.Stops[id]
	if !ok {
		stop, ok = data.Stops[stationStopID(id)]
	}
	if !ok || stop.Longitude == 0 || stop.Latitude == 0 {
		return nil
	}
	return []float64{stop.Longitude, stop.Latitude}
}

func (s *StaticService) StationMap(ctx context.Context, stopID string, now time.Time) (StationMapResponse, error) {
	data, err := s.load(ctx)
	if err != nil {
		return StationMapResponse{}, err
	}
	entities, err := s.realtime.Entities(ctx)
	if err != nil {
		return StationMapResponse{}, err
	}
	bootstrap := buildBootstrap(data, realtimeRoutesByStop(entities, now), now)
	var station Station
	for _, item := range bootstrap.Stations {
		for _, service := range item.Services {
			if service.ID == stationStopID(stopID) {
				station = item
				break
			}
		}
	}
	if station.ID == "" {
		return StationMapResponse{}, fmt.Errorf("station %s not found", stopID)
	}
	routeIDs := mapStationRoutes(station)
	displayed, serviceStops := map[string]struct{}{}, map[string]struct{}{}
	for _, route := range routeIDs {
		displayed[route] = struct{}{}
	}
	for _, service := range station.Services {
		serviceStops[service.ID] = struct{}{}
	}
	lines := realtimeStationLines(entities, data, routeIDs, serviceStops)
	found := map[string]bool{}
	for _, line := range lines {
		found[line.Properties.RouteID] = true
	}
	candidates := []staticTrip{}
	for _, trip := range data.Trips {
		if _, ok := displayed[routeFamily(trip.RouteID)]; !ok {
			continue
		}
		for _, stop := range data.StopTimesByTrip[trip.ID] {
			if _, ok := serviceStops[stationStopID(stop.StopID)]; ok {
				candidates = append(candidates, trip)
				break
			}
		}
	}
	fallback := routeLineFeatures(candidates, data.ShapesByID, activeServiceIDs(data, now))
	for _, route := range routeIDs {
		if found[route] {
			continue
		}
		for _, line := range fallback {
			if line.Properties.RouteID == route {
				line.Geometry.Coordinates = smoothMapCoordinates(line.Geometry.Coordinates, 3)
				lines = append(lines, line)
				break
			}
		}
	}
	for index := range lines {
		lines[index].Properties.Offset = (float64(index) - float64(len(lines)-1)/2) * 8
	}
	return StationMapResponse{Lines: lines, Station: station, Vehicles: mapVehicles(entities, data, displayed, lines, now)}, nil
}

func realtimeStationLines(entities []*gtfs.FeedEntity, data *staticData, routes []string, stationIDs map[string]struct{}) []MapFeature {
	type path struct {
		coordinates   [][]float64
		authoritative bool
	}
	displayed := map[string]bool{}
	for _, route := range routes {
		displayed[routeFamily(route)] = true
	}
	candidates := map[string]path{}
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		route := routeFamily(update.GetTrip().GetRouteId())
		if !displayed[route] {
			continue
		}
		serves := false
		for _, stop := range update.GetStopTimeUpdate() {
			if _, ok := stationIDs[stationStopID(stop.GetStopId())]; ok {
				serves = true
				break
			}
		}
		if !serves {
			continue
		}
		trip := data.TripsByRealtimeID[tripLookupID(update.GetTrip().GetTripId())]
		coordinates := shapeCoordinates(data.ShapesByID[trip.ShapeID])
		authoritative := len(coordinates) >= 2
		if !authoritative {
			coordinates = nil
			for _, stop := range update.GetStopTimeUpdate() {
				point := stopCoordinate(stop.GetStopId(), data)
				if point != nil && (len(coordinates) == 0 || point[0] != coordinates[len(coordinates)-1][0] || point[1] != coordinates[len(coordinates)-1][1]) {
					coordinates = append(coordinates, point)
				}
			}
		}
		if len(coordinates) >= 2 && len(coordinates) > len(candidates[route].coordinates) {
			candidates[route] = path{coordinates, authoritative}
		}
	}
	lines := []MapFeature{}
	for _, route := range routes {
		candidate, ok := candidates[routeFamily(route)]
		if !ok {
			continue
		}
		coordinates := candidate.coordinates
		if !candidate.authoritative {
			coordinates = smoothMapCoordinates(coordinates, 8)
		}
		line := MapFeature{Type: "Feature"}
		line.Properties.RouteID = routeFamily(route)
		line.Properties.Color = "#" + routeColors[line.Properties.RouteID]
		line.Geometry.Type = "LineString"
		line.Geometry.Coordinates = coordinates
		lines = append(lines, line)
	}
	for index := range lines {
		lines[index].Properties.Offset = (float64(index) - float64(len(lines)-1)/2) * 8
	}
	return lines
}

func smoothMapCoordinates(coordinates [][]float64, steps int) [][]float64 {
	if len(coordinates) < 3 {
		return coordinates
	}
	result := [][]float64{coordinates[0]}
	for index := 0; index < len(coordinates)-1; index++ {
		p0, p1 := coordinates[max(0, index-1)], coordinates[index]
		p2, p3 := coordinates[index+1], coordinates[min(len(coordinates)-1, index+2)]
		for step := 1; step <= steps; step++ {
			t := float64(step) / float64(steps)
			t2, t3 := t*t, t*t*t
			point := make([]float64, 2)
			for axis := range 2 {
				point[axis] = 0.5 * (2*p1[axis] + (-p0[axis]+p2[axis])*t + (2*p0[axis]-5*p1[axis]+4*p2[axis]-p3[axis])*t2 + (-p0[axis]+3*p1[axis]-3*p2[axis]+p3[axis])*t3)
			}
			result = append(result, point)
		}
	}
	return result
}

func mapVehicles(entities []*gtfs.FeedEntity, data *staticData, displayed map[string]struct{}, lines []MapFeature, now time.Time) []MapVehicle {
	vehicles := []MapVehicle{}
	reported := map[string]*gtfs.Position{}
	for _, entity := range entities {
		if vehicle := entity.GetVehicle(); vehicle != nil && vehicle.GetPosition() != nil {
			reported[vehicle.GetTrip().GetTripId()] = vehicle.GetPosition()
		}
	}
	for _, entity := range entities {
		update := entity.GetTripUpdate()
		route := routeFamily(update.GetTrip().GetRouteId())
		if _, ok := displayed[route]; !ok || !tripHasDeparted(update, data, now) {
			continue
		}
		type point struct {
			coordinate []float64
			timestamp  int64
		}
		points := []point{}
		for _, item := range update.GetStopTimeUpdate() {
			timestamp := eventTimestamp(item)
			coordinate := stopCoordinate(item.GetStopId(), data)
			if timestamp > 0 && coordinate != nil {
				points = append(points, point{coordinate, timestamp})
			}
		}
		next := -1
		for index, item := range points {
			if item.timestamp >= now.Unix() {
				next = index
				break
			}
		}
		if next < 0 || len(points) == 0 {
			continue
		}
		coordinate := append([]float64(nil), points[next].coordinate...)
		bearing := float64(0)
		if next > 0 {
			previous, following := points[next-1], points[next]
			progress := math.Max(0, math.Min(1, float64(now.Unix()-previous.timestamp)/float64(max(1, following.timestamp-previous.timestamp))))
			coordinate[0] = previous.coordinate[0] + (following.coordinate[0]-previous.coordinate[0])*progress
			coordinate[1] = previous.coordinate[1] + (following.coordinate[1]-previous.coordinate[1])*progress
			bearing = coordinateBearing(previous.coordinate, following.coordinate)
		} else if len(points) > 1 {
			bearing = coordinateBearing(points[0].coordinate, points[1].coordinate)
		}
		if position := reported[update.GetTrip().GetTripId()]; position != nil {
			coordinate = []float64{float64(position.GetLongitude()), float64(position.GetLatitude())}
			if position.Bearing != nil {
				bearing = float64(position.GetBearing())
			}
		}
		vehicles = append(vehicles, MapVehicle{ID: update.GetTrip().GetTripId(), RouteID: route, Longitude: coordinate[0], Latitude: coordinate[1], DirectionBearing: bearing})
	}
	for index := range vehicles {
		vehicle := &vehicles[index]
		color := "#" + routeColors[routeFamily(vehicle.RouteID)]
		latitudeScale := math.Cos(vehicle.Latitude * math.Pi / 180)
		bestDistance := math.Inf(1)
		originalLongitude, originalLatitude := vehicle.Longitude, vehicle.Latitude
		for _, line := range lines {
			if line.Properties.Color != color {
				continue
			}
			for pointIndex := 1; pointIndex < len(line.Geometry.Coordinates); pointIndex++ {
				start, end := line.Geometry.Coordinates[pointIndex-1], line.Geometry.Coordinates[pointIndex]
				startX, endX, vehicleX := start[0]*latitudeScale, end[0]*latitudeScale, originalLongitude*latitudeScale
				dx, dy := endX-startX, end[1]-start[1]
				length := dx*dx + dy*dy
				if length == 0 {
					continue
				}
				progress := math.Max(0, math.Min(1, ((vehicleX-startX)*dx+(originalLatitude-start[1])*dy)/length))
				longitude := start[0] + (end[0]-start[0])*progress
				latitude := start[1] + (end[1]-start[1])*progress
				distance := math.Pow((originalLongitude-longitude)*latitudeScale, 2) + math.Pow(originalLatitude-latitude, 2)
				if distance < bestDistance {
					bestDistance, vehicle.Longitude, vehicle.Latitude = distance, longitude, latitude
					vehicle.Bearing = math.Atan2(dx, dy) * 180 / math.Pi
					vehicle.Offset = line.Properties.Offset
				}
			}
		}
	}
	return vehicles
}
