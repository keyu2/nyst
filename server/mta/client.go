package mta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/keyu2/nyst/server/cache"
)

const alertsURL = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/camsys%2Fsubway-alerts.json"

type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
	alerts     []alertEntity
	loadedAt   time.Time
}

type alertFeed struct {
	Entities []alertEntity `json:"entity"`
}
type translatedText struct {
	Translation []struct{ Text, Language string } `json:"translation"`
}
type alertEntity struct {
	ID    string `json:"id"`
	Alert struct {
		ActivePeriod   []struct{ Start, End int64 } `json:"active_period"`
		InformedEntity []struct {
			RouteID string `json:"route_id"`
		} `json:"informed_entity"`
		Cause           int            `json:"cause"`
		HeaderText      translatedText `json:"header_text"`
		DescriptionText translatedText `json:"description_text"`
		Mercury         struct {
			AlertType                 string         `json:"alert_type"`
			HumanReadableActivePeriod translatedText `json:"human_readable_active_period"`
		} `json:"transit_realtime.mercury_alert"`
	} `json:"alert"`
}
type RouteAlert struct {
	ID          string `json:"id"`
	Header      string `json:"header"`
	Description string `json:"description"`
	AlertType   string `json:"alertType"`
	TimeLabel   string `json:"timeLabel"`
	Category    string `json:"category"`
}
type RouteAlertSnapshot struct {
	RouteAlerts   map[string][]RouteAlert `json:"routeAlerts"`
	RouteStatuses map[string]string       `json:"routeStatuses"`
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 6 * time.Second}}
}

func (c *Client) AlertCount(ctx context.Context) (int, error) {
	entities, err := c.alertEntities(ctx)
	return len(entities), err
}

func (c *Client) alertEntities(ctx context.Context) ([]alertEntity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.alerts) > 0 && time.Since(c.loadedAt) < 30*time.Second {
		return c.alerts, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, alertsURL, nil)
	if err != nil {
		return c.alerts, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return c.alerts, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return c.alerts, fmt.Errorf("MTA alerts returned %s", response.Status)
	}

	var feed alertFeed
	if err := json.NewDecoder(response.Body).Decode(&feed); err != nil {
		return c.alerts, fmt.Errorf("decode MTA alerts: %w", err)
	}
	c.alerts, c.loadedAt = feed.Entities, time.Now()
	return c.alerts, nil
}

var delayPattern = regexp.MustCompile(`(?i)\b(delay|delays|delayed)\b`)
var suspensionPattern = regexp.MustCompile(`(?i)\b(part(?:ially)? suspended|suspended|no (?:\[[^\]]+\]|\S+) service|service suspended)\b`)
var schedulePattern = regexp.MustCompile(`\b(saturday|sunday|weekend|holiday|special) schedule\b`)

func english(value translatedText) string {
	for _, item := range value.Translation {
		if item.Language == "en" {
			return item.Text
		}
	}
	for _, item := range value.Translation {
		if strings.HasPrefix(item.Language, "en") {
			return item.Text
		}
	}
	if len(value.Translation) > 0 {
		return value.Translation[0].Text
	}
	return ""
}
func plannedAlert(cause int, alertType string) bool {
	lower := strings.ToLower(strings.TrimSpace(alertType))
	return cause == 9 || cause == 10 || strings.HasPrefix(lower, "planned") || lower == "no scheduled service" || strings.Contains(lower, "station notice") || schedulePattern.MatchString(lower)
}
func alertStatus(alert RouteAlert) string {
	if alert.Category == "planned" {
		return "planned"
	}
	if delayPattern.MatchString(alert.AlertType) {
		return "delay"
	}
	if suspensionPattern.MatchString(alert.AlertType) || suspensionPattern.MatchString(alert.Header+" "+alert.Description) {
		return "suspended"
	}
	return "delay"
}
func (c *Client) RouteAlertSnapshot(ctx context.Context, routeIDs []string, now time.Time) (RouteAlertSnapshot, error) {
	entities, err := c.alertEntities(ctx)
	if err != nil && len(entities) == 0 {
		return RouteAlertSnapshot{}, err
	}
	requested := map[string]string{}
	snapshot := RouteAlertSnapshot{RouteAlerts: map[string][]RouteAlert{}, RouteStatuses: map[string]string{}}
	for _, id := range routeIDs {
		requested[normalizeRouteID(id)] = id
		snapshot.RouteAlerts[id] = []RouteAlert{}
	}
	severity := map[string]int{"planned": 1, "delay": 2, "suspended": 3}
	for _, entity := range entities {
		active := len(entity.Alert.ActivePeriod) == 0
		for _, period := range entity.Alert.ActivePeriod {
			if (period.Start == 0 || period.Start <= now.Unix()) && (period.End == 0 || period.End >= now.Unix()) {
				active = true
				break
			}
		}
		if !active {
			continue
		}
		routes := map[string]struct{}{}
		for _, informed := range entity.Alert.InformedEntity {
			if id := requested[normalizeRouteID(informed.RouteID)]; id != "" {
				routes[id] = struct{}{}
			}
		}
		if len(routes) == 0 {
			continue
		}
		alertType := entity.Alert.Mercury.AlertType
		category := "happening"
		if plannedAlert(entity.Alert.Cause, alertType) {
			category = "planned"
		}
		alert := RouteAlert{ID: entity.ID, Header: english(entity.Alert.HeaderText), Description: english(entity.Alert.DescriptionText), AlertType: alertType, TimeLabel: english(entity.Alert.Mercury.HumanReadableActivePeriod), Category: category}
		if alert.Header == "" && alert.Description == "" {
			continue
		}
		status := alertStatus(alert)
		for id := range routes {
			snapshot.RouteAlerts[id] = append(snapshot.RouteAlerts[id], alert)
			if severity[status] > severity[snapshot.RouteStatuses[id]] {
				snapshot.RouteStatuses[id] = status
			}
		}
	}
	return snapshot, err
}

func (c *Client) RefreshAlerts(ctx context.Context, store *cache.Store, interval time.Duration) {
	if interval <= 0 {
		panic("refresh interval must be positive")
	}

	refresh := func() {
		count, err := c.AlertCount(ctx)
		now := time.Now().UTC()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				store.Fail(err, now)
			}
			return
		}
		store.Publish(count, now)
	}

	refresh()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}
