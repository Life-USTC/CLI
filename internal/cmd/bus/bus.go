package bus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdBus() *cobra.Command {
	var (
		origin, destination, dayType, now string
		showDeparted, includeAll          bool
		limit                             int
	)
	cmd := &cobra.Command{
		Use:   "bus [command]",
		Short: "Shuttle bus schedules",
		Long:  "Query shuttle bus schedules between campuses, and manage your bus preferences.",
		Example: `  # Query upcoming buses
  life-ustc bus

  # Filter by route
  life-ustc bus query --from 1 --to 2

  # Show departed trips too
  life-ustc bus --show-departed

  # Show saved preferences
  life-ustc bus preferences`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBusQuery(cmd, origin, destination, dayType, now, showDeparted, includeAll, limit)
		},
	}
	addBusQueryFlags(cmd, &origin, &destination, &dayType, &now, &showDeparted, &includeAll, &limit)
	cmd.AddCommand(newCmdQuery())
	cmd.AddCommand(newCmdPreferences())
	cmd.AddCommand(newCmdSetPreferences())
	return cmd
}

func addBusQueryFlags(cmd *cobra.Command, origin, destination, dayType, now *string, showDeparted, includeAll *bool, limit *int) {
	cmd.Flags().StringVar(origin, "from", "", "Origin campus ID")
	cmd.Flags().StringVar(destination, "to", "", "Destination campus ID")
	cmd.Flags().StringVar(dayType, "day-type", "", "Day type: auto, weekday, weekend")
	cmd.Flags().StringVar(now, "now", "", "Override current time (ISO 8601)")
	cmd.Flags().BoolVar(showDeparted, "show-departed", false, "Show already-departed trips")
	cmd.Flags().BoolVar(includeAll, "all", false, "Include all trips (not just upcoming)")
	cmd.Flags().IntVar(limit, "limit", 0, "Max trips to show")
}

func runBusQuery(cmd *cobra.Command, origin, destination, dayType, now string, showDeparted, includeAll bool, limit int) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := &openapi.QueryBusParams{}
	if origin != "" {
		params.OriginCampusId = &origin
	}
	if destination != "" {
		params.DestinationCampusId = &destination
	}
	if dayType != "" {
		dt := openapi.QueryBusParamsDayType(dayType)
		params.DayType = &dt
	}
	if now != "" {
		t, err := time.Parse(time.RFC3339, now)
		if err != nil {
			return fmt.Errorf("invalid time format (expected RFC 3339): %w", err)
		}
		params.Now = &t
	}
	if showDeparted {
		v := openapi.QueryBusParamsShowDepartedTripsTrue
		params.ShowDepartedTrips = &v
	}
	if includeAll {
		v := openapi.QueryBusParamsIncludeAllTripsTrue
		params.IncludeAllTrips = &v
	}
	if limit > 0 {
		l := cmdutil.Itoa(limit)
		params.Limit = &l
	}
	data, err := api.ParseResponseRaw(c.QueryBus(api.Ctx(), params))
	if err != nil {
		return err
	}
	if output.IsJSON() {
		output.JSON(data)
		return nil
	}
	renderBus(cmdutil.AsMap(data))
	return nil
}

func newCmdQuery() *cobra.Command {
	var (
		origin, destination, dayType, now string
		showDeparted, includeAll          bool
		limit                             int
	)
	cmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Query shuttle bus schedules",
		Example: `  # Show all upcoming buses
  life-ustc bus query

  # Filter by origin and destination campus ID
  life-ustc bus query --from 1 --to 2

  # Show departed trips
  life-ustc bus query --show-departed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBusQuery(cmd, origin, destination, dayType, now, showDeparted, includeAll, limit)
		},
	}
	addBusQueryFlags(cmd, &origin, &destination, &dayType, &now, &showDeparted, &includeAll, &limit)
	return cmd
}

func renderBus(data map[string]any) {
	if data == nil {
		output.Dim("  No bus schedules found.")
		return
	}

	rec := cmdutil.AsMap(data["recommended"])
	if rec != nil {
		routeName := resolveRouteName(rec)
		fmt.Println()
		output.Bold(fmt.Sprintf("  ★ Recommended — %s", routeName))
		printRouteMatch(rec)
	}

	if matches, ok := data["matches"].([]any); ok {
		for _, m := range matches {
			match := cmdutil.AsMap(m)
			if match == nil {
				continue
			}
			if isRec, _ := match["isRecommended"].(bool); isRec && rec != nil {
				continue
			}
			routeName := resolveRouteName(match)
			fmt.Println()
			output.Bold(fmt.Sprintf("  %s", routeName))
			printRouteMatch(match)
		}
	}

	if rec == nil {
		if matches, _ := data["matches"].([]any); len(matches) == 0 {
			output.Dim("  No bus schedules found.")
		}
	}

	if notice := cmdutil.AsMap(data["notice"]); notice != nil {
		if content, ok := notice["content"].(string); ok && content != "" {
			fmt.Println()
			output.Dim(fmt.Sprintf("  Notice: %s", content))
		}
	}
}

func resolveRouteName(match map[string]any) string {
	if route := cmdutil.AsMap(match["route"]); route != nil {
		if name, ok := route["nameCn"].(string); ok {
			return name
		}
	}
	return "Route"
}

func printRouteMatch(match map[string]any) {
	nextTrip := cmdutil.AsMap(match["nextTrip"])
	upcoming, _ := match["upcomingTrips"].([]any)
	totalTrips := 0
	if t, ok := match["totalTrips"].(float64); ok {
		totalTrips = int(t)
	}

	if nextTrip != nil {
		label := ""
		if mins, ok := nextTrip["minutesUntilDeparture"].(float64); ok {
			label = fmt.Sprintf("in %dmin", int(mins))
		}
		printTripLine(nextTrip, true, label)
	}

	nextID := ""
	if nextTrip != nil {
		if id, ok := nextTrip["id"].(string); ok {
			nextID = id
		}
	}

	for _, t := range upcoming {
		trip := cmdutil.AsMap(t)
		if trip == nil {
			continue
		}
		if id, ok := trip["id"].(string); ok && id == nextID {
			continue
		}
		printTripLine(trip, false, "")
	}

	shown := len(upcoming)
	if nextTrip != nil {
		shown++
	}
	if totalTrips > shown {
		output.Dim(fmt.Sprintf("    … and %d more trips", totalTrips-shown))
	}
}

func printTripLine(trip map[string]any, highlight bool, label string) {
	dep, _ := trip["departureTime"].(string)
	arr, _ := trip["arrivalTime"].(string)
	stops, _ := trip["stopTimes"].([]any)

	var names []string
	for _, s := range stops {
		st := cmdutil.AsMap(s)
		if st == nil {
			continue
		}
		if pass, _ := st["isPassThrough"].(bool); pass {
			continue
		}
		if name, ok := st["campusName"].(string); ok && name != "" {
			names = append(names, name)
		}
	}

	timeStr := dep
	if dep != "" && arr != "" {
		timeStr = dep + " → " + arr
	}

	line := fmt.Sprintf("    %s", timeStr)
	if len(names) > 0 {
		line += fmt.Sprintf("  (%s)", joinStrings(names, " → "))
	}
	if label != "" {
		line += "  " + color.GreenString(label)
	}

	if highlight {
		fmt.Println(color.New(color.Bold).Sprint(line))
	} else {
		fmt.Println(line)
	}
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func newCmdPreferences() *cobra.Command {
	return &cobra.Command{
		Use:   "preferences",
		Aliases: []string{"prefs"},
		Short: "Show your bus preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := api.ParseResponseRaw(c.GetBusPreferences(api.Ctx()))
			if err != nil {
				return err
			}
			output.OutputDetail(data, []output.FieldDef{
				{Key: "preferredOriginCampusId", Label: "Preferred origin"},
				{Key: "preferredDestinationCampusId", Label: "Preferred destination"},
				{Key: "favoriteCampusIds", Label: "Favorite campuses"},
				{Key: "favoriteRouteIds", Label: "Favorite routes"},
				{Key: "showDepartedTrips", Label: "Show departed"},
			}, "Bus preferences")
			return nil
		},
	}
}

func newCmdSetPreferences() *cobra.Command {
	var (
		origin, destination int
		showDeparted        bool
		rawJSON             string
	)
	cmd := &cobra.Command{
		Use:   "set-preferences",
		Short: "Update bus preferences",
		Example: `  # Set preferred origin and destination
  life-ustc bus set-preferences --origin 1 --destination 2

  # Toggle show departed trips
  life-ustc bus set-preferences --show-departed

  # Set from raw JSON
  life-ustc bus set-preferences --json '{"showDepartedTrips":true}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			var body map[string]any
			if rawJSON != "" {
				if err := json.Unmarshal([]byte(rawJSON), &body); err != nil {
					return fmt.Errorf("invalid JSON: %w", err)
				}
			} else {
				body = map[string]any{}
				if cmd.Flags().Changed("origin") {
					body["preferredOriginCampusId"] = origin
				}
				if cmd.Flags().Changed("destination") {
					body["preferredDestinationCampusId"] = destination
				}
				if cmd.Flags().Changed("show-departed") {
					body["showDepartedTrips"] = showDeparted
				}
				if len(body) == 0 {
					return fmt.Errorf("specify at least one flag (--origin, --destination, --show-departed) or use --json")
				}
			}
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to encode body: %w", err)
			}
			_, err = api.ParseResponseRaw(c.SetBusPreferencesWithBody(api.Ctx(), "application/json", bytes.NewReader(bodyBytes)))
			if err != nil {
				return err
			}
			output.Success("Bus preferences updated.")
			return nil
		},
	}
	cmd.Flags().IntVar(&origin, "origin", 0, "Preferred origin campus ID")
	cmd.Flags().IntVar(&destination, "destination", 0, "Preferred destination campus ID")
	cmd.Flags().BoolVar(&showDeparted, "show-departed", false, "Show departed trips by default")
	cmd.Flags().StringVar(&rawJSON, "raw-json", "", "Set preferences from raw JSON body")
	return cmd
}
