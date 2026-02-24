package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"mpr/internal/models"

	"github.com/spf13/cobra"
)

func reportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reports",
		Short: "Discover available reports",
	}

	cmd.AddCommand(reportsListCmd())
	cmd.AddCommand(reportsSearchCmd())
	cmd.AddCommand(reportsShowCmd())
	return cmd
}

func reportsListCmd() *cobra.Command {
	var (
		jsonOut    bool
		tableOut   bool
		market     string
		office     string
		marketType string
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List published reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}

			reports, _, err := loadReports(context.Background(), cfg, refresh)
			if err != nil {
				return err
			}

			filtered := filterReports(reports, market, office, marketType)
			if jsonOut && tableOut {
				return fmt.Errorf("use either --json or --table")
			}

			if jsonOut {
				data, err := json.MarshalIndent(filtered, "", "  ")
				if err != nil {
					return err
				}
				writeJSON(data)
				return nil
			}

			if !tableOut && !jsonOut {
				tableOut = true
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "slug_id\tslug_name\treport_title\tpublished_date")
			for _, report := range filtered {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", report.SlugID, report.SlugName, report.ReportTitle, report.PublishedDate)
			}
			return w.Flush()
		},
	}
	_ = cmd.RegisterFlagCompletionFunc("market", func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return listValueCompletion(cmd, toComplete, func(report models.ReportItem) []string {
			return report.Markets
		})
	})
	_ = cmd.RegisterFlagCompletionFunc("office", func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return listValueCompletion(cmd, toComplete, func(report models.ReportItem) []string {
			return report.Offices
		})
	})
	_ = cmd.RegisterFlagCompletionFunc("market-type", func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return listValueCompletion(cmd, toComplete, func(report models.ReportItem) []string {
			return report.MarketTypes
		})
	})

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&tableOut, "table", false, "Output table")
	cmd.Flags().StringVar(&market, "market", "", "Filter by market")
	cmd.Flags().StringVar(&office, "office", "", "Filter by office")
	cmd.Flags().StringVar(&marketType, "market-type", "", "Filter by market type")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh cached reports")
	return cmd
}

func reportsSearchCmd() *cobra.Command {
	var (
		regex   bool
		refresh bool
	)

	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search report titles",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}

			reports, _, err := loadReports(context.Background(), cfg, refresh)
			if err != nil {
				return err
			}

			term := args[0]
			matches, err := searchReports(reports, term, regex)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(matches, "", "  ")
			if err != nil {
				return err
			}
			writeJSON(data)
			return nil
		},
	}

	cmd.Flags().BoolVar(&regex, "regex", false, "Treat search text as regex")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh cached reports")
	return cmd
}

func reportsShowCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "show <slug_id>",
		Short: "Show metadata for a report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}

			reports, _, err := loadReports(context.Background(), cfg, refresh)
			if err != nil {
				return err
			}

			slugID := strings.TrimSpace(args[0])
			report, ok := findReport(reports, slugID)
			if !ok {
				return fmt.Errorf("slug_id not found: %s", slugID)
			}

			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			writeJSON(data)
			return nil
		},
	}
	cmd.ValidArgsFunction = slugCompletion

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh cached reports")
	return cmd
}

func filterReports(reports []models.ReportItem, market, office, marketType string) []models.ReportItem {
	market = strings.TrimSpace(strings.ToLower(market))
	office = strings.TrimSpace(strings.ToLower(office))
	marketType = strings.TrimSpace(strings.ToLower(marketType))

	filtered := make([]models.ReportItem, 0, len(reports))
	for _, report := range reports {
		if market != "" && !containsFold(report.Markets, market) {
			continue
		}
		if office != "" && !containsFold(report.Offices, office) {
			continue
		}
		if marketType != "" && !containsFold(report.MarketTypes, marketType) {
			continue
		}
		filtered = append(filtered, report)
	}
	return filtered
}

func searchReports(reports []models.ReportItem, term string, useRegex bool) ([]models.ReportItem, error) {
	matches := make([]models.ReportItem, 0)
	if useRegex {
		re, err := regexp.Compile(term)
		if err != nil {
			return nil, err
		}
		for _, report := range reports {
			if re.MatchString(report.ReportTitle) || re.MatchString(report.SlugName) {
				matches = append(matches, report)
			}
		}
		return matches, nil
	}

	term = strings.ToLower(term)
	for _, report := range reports {
		if strings.Contains(strings.ToLower(report.ReportTitle), term) || strings.Contains(strings.ToLower(report.SlugName), term) {
			matches = append(matches, report)
		}
	}
	return matches, nil
}

func findReport(reports []models.ReportItem, slugID string) (models.ReportItem, bool) {
	for _, report := range reports {
		if fmt.Sprintf("%d", report.SlugID) == slugID {
			return report, true
		}
	}
	return models.ReportItem{}, false
}

func containsFold(values []string, needle string) bool {
	for _, v := range values {
		if strings.ToLower(v) == needle {
			return true
		}
	}
	return false
}
