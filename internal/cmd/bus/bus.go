package bus

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdBus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bus <command>",
		Short: "Shuttle bus schedules",
	}
	cmd.AddCommand(newCmdQuery())
	cmd.AddCommand(newCmdPreferences())
	cmd.AddCommand(newCmdSetPreferences())
	return cmd
}

func newCmdQuery() *cobra.Command {
	var (
		origin, destination, dayType, now string
		limit                             int
	)
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query shuttle bus schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{}
			if origin != "" {
				params.Set("from", origin)
			}
			if destination != "" {
				params.Set("to", destination)
			}
			if dayType != "" {
				params.Set("dayType", dayType)
			}
			if now != "" {
				params.Set("now", now)
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/bus", params)
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			renderBus(cmdutil.AsMap(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&origin, "from", "", "Origin campus")
	cmd.Flags().StringVar(&destination, "to", "", "Destination campus")
	cmd.Flags().StringVar(&dayType, "day-type", "", "Day type filter")
	cmd.Flags().StringVar(&now, "now", "", "Override current time (ISO 8601)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max trips")
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
		Short: "Show your bus preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/bus/preferences", nil)
			if err != nil {
				return err
			}
			output.OutputDetail(data, []output.FieldDef{
				{Key: "defaultFrom", Label: "Default from"},
				{Key: "defaultTo", Label: "Default to"},
				{Key: "notifications", Label: "Notifications"},
			}, "Bus preferences")
			return nil
		},
	}
}

func newCmdSetPreferences() *cobra.Command {
	return &cobra.Command{
		Use:   "set-preferences <json>",
		Short: "Update bus preferences (pass JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(args[0]), &body); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			_, err = c.Post("/api/bus/preferences", body)
			if err != nil {
				return err
			}
			output.Success("Bus preferences updated.")
			return nil
		},
	}
}
