package catalog

import (
	"sort"
	"strings"
)

// CalculateRelevanceScore evaluates how closely an application matches a user search query.
// It prioritizes exact name and AppID matches, followed by prefix matches, aliases, word boundaries,
// installed status, and official repository distribution.
func CalculateRelevanceScore(app UnifiedApplication, query string) float64 {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery == "" {
		return 0
	}
	terms := strings.Fields(trimmedQuery)
	if len(terms) == 0 {
		return 0
	}

	appName := strings.ToLower(strings.TrimSpace(app.Name))
	appID := strings.ToLower(strings.TrimSpace(app.AppID))
	appSummary := strings.ToLower(strings.TrimSpace(app.Summary))
	appDesc := strings.ToLower(strings.TrimSpace(app.Description))

	var score float64

	// 1. Exact match on Name or AppID (Highest signal)
	if appName == trimmedQuery || appID == trimmedQuery {
		score += 3500
	} else if len(terms) > 1 && (appName == terms[0] || appID == terms[0]) {
		// Searching "firefox browser" directly matches application "firefox"
		score += 2500
	}

	// 2. Prefix match on Name or AppID
	if strings.HasPrefix(appName, trimmedQuery) || strings.HasPrefix(appID, trimmedQuery) {
		score += 1500
	} else if strings.HasPrefix(appName, terms[0]) || strings.HasPrefix(appID, terms[0]) {
		score += 1000
	}

	// 3. Match against package aliases (e.g. "org.mozilla.firefox", "firefox")
	for _, alias := range app.Aliases {
		lowAlias := strings.ToLower(alias)
		if lowAlias == trimmedQuery {
			score += 3000
			break
		} else if len(terms) > 1 && lowAlias == terms[0] {
			score += 2000
			break
		} else if strings.HasPrefix(lowAlias, trimmedQuery) {
			score += 800
			break
		} else if strings.HasPrefix(lowAlias, terms[0]) {
			score += 600
			break
		}
	}

	// 4. Term matches in Name and AppID
	matchedTermsInName := 0
	for _, t := range terms {
		if strings.Contains(appName, t) || strings.Contains(appID, t) {
			matchedTermsInName++
			score += 350
		}
	}
	if len(terms) > 1 && matchedTermsInName == len(terms) {
		score += 600 // All search terms found in name
	}

	// 5. Matches in Summary & Description
	matchedTermsInDesc := 0
	for _, t := range terms {
		if strings.Contains(appSummary, t) || strings.Contains(appDesc, t) {
			matchedTermsInDesc++
			score += 50
		}
	}
	if len(terms) > 1 && matchedTermsInDesc == len(terms) {
		score += 120 // All search terms found in description
	}

	// 6. Installed & Quality boost
	if app.IsInstalled {
		score += 250
	}

	// Distribution origin quality boost: Native official repo > Flatpak/Snap > Third-party
	for _, o := range app.Offers {
		prov := strings.ToLower(o.Provider)
		if prov == "pacman" || prov == "repository" || prov == "apt" || prov == "dnf" || prov == "zypper" {
			score += 150
		}
		if o.Trust.OfficialRepo {
			score += 100
		}
		if o.RecommendationBadge == "Recommended" {
			score += 50
		}
	}

	return score
}

// SortByRelevance orders a slice of UnifiedApplications in descending order of query relevance.
// In case of score ties, it falls back to alphabetical order by Name for deterministic stability.
func SortByRelevance(apps []UnifiedApplication, query string) {
	if len(apps) <= 1 || strings.TrimSpace(query) == "" {
		return
	}

	scores := make(map[string]float64, len(apps))
	for _, app := range apps {
		scores[app.AppID] = CalculateRelevanceScore(app, query)
	}

	sort.SliceStable(apps, func(i, j int) bool {
		sI := scores[apps[i].AppID]
		sJ := scores[apps[j].AppID]
		if sI != sJ {
			return sI > sJ
		}
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})
}

