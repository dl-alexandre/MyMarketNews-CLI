package commands

import (
	"fmt"
	"strings"
	"sync"

	"mpr/internal/cache"
	"mpr/internal/models"

	"github.com/spf13/cobra"
)

var completionCache struct {
	once    sync.Once
	reports []models.ReportItem
}

func loadCompletionReports(cmd *cobra.Command) []models.ReportItem {
	completionCache.once.Do(func() {
		cacheDir, _ := cmd.Flags().GetString("cache-dir")
		reports, ok, err := cache.LoadReportsFromCache(cacheDir)
		if err != nil || !ok {
			completionCache.reports = nil
			return
		}
		completionCache.reports = reports
	})

	return completionCache.reports
}

func slugCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	reports := loadCompletionReports(cmd)
	if len(reports) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(reports))
	for _, report := range reports {
		slug := fmt.Sprintf("%d", report.SlugID)
		if toComplete != "" && !strings.HasPrefix(slug, toComplete) {
			continue
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		if report.ReportTitle != "" {
			out = append(out, fmt.Sprintf("%s\t%s", slug, report.ReportTitle))
		} else {
			out = append(out, slug)
		}
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func sectionCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	slug := strings.TrimSpace(args[0])
	if slug == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	reports := loadCompletionReports(cmd)
	if len(reports) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	for _, report := range reports {
		if fmt.Sprintf("%d", report.SlugID) != slug {
			continue
		}
		sections := make([]string, 0, len(report.SectionNames))
		for _, section := range report.SectionNames {
			if toComplete != "" && !strings.HasPrefix(strings.ToLower(section), strings.ToLower(toComplete)) {
				continue
			}
			sections = append(sections, section)
		}
		return sections, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func listValueCompletion(cmd *cobra.Command, toComplete string, extract func(models.ReportItem) []string) ([]string, cobra.ShellCompDirective) {
	reports := loadCompletionReports(cmd)
	if len(reports) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	seen := map[string]bool{}
	out := []string{}
	for _, report := range reports {
		for _, value := range extract(report) {
			candidate := strings.TrimSpace(value)
			if candidate == "" {
				continue
			}
			if toComplete != "" && !strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(toComplete)) {
				continue
			}
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			out = append(out, candidate)
		}
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}
