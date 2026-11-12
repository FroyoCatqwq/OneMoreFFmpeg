package provider

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

var stableLinePattern = regexp.MustCompile(`^n[0-9]+\.[0-9]+$`)

type assetQuery struct {
	Channel     string
	Target      string
	Variant     string
	Linkage     string
	ReleaseLine string
}

type parsedAsset struct {
	asset     github.Asset
	target    string
	variant   string
	linkage   string
	line      string
	extension string
}

// normalizeVariant folds the documented "-shared" variants into a variant plus
// linkage pair, e.g. nonfree-shared -> (nonfree, shared).
func normalizeVariant(variant, linkage string) (string, string) {
	normalized := strings.ToLower(strings.TrimSpace(variant))
	if strings.HasSuffix(normalized, "-shared") {
		normalized = strings.TrimSuffix(normalized, "-shared")
		linkage = "shared"
	}
	if normalized == "" {
		normalized = "gpl"
	}
	if linkage == "" {
		linkage = "static"
	}
	return normalized, strings.ToLower(linkage)
}

func parseAssetName(name string) (parsedAsset, bool) {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "checksums") || !strings.HasPrefix(lower, "ffmpeg-") {
		return parsedAsset{}, false
	}
	extension := ""
	switch {
	case strings.HasSuffix(lower, ".tar.xz"):
		extension = ".tar.xz"
	case strings.HasSuffix(lower, ".zip"):
		extension = ".zip"
	default:
		return parsedAsset{}, false
	}

	tokens := strings.Split(strings.TrimSuffix(lower, extension), "-")
	parsed := parsedAsset{asset: github.Asset{Name: name}, linkage: "static", extension: extension}
	for _, token := range tokens {
		switch {
		case token == "win64" || token == "linux64":
			parsed.target = token
		case token == "gpl" || token == "lgpl" || token == "nonfree":
			parsed.variant = token
		case token == "shared":
			parsed.linkage = "shared"
		case stableLinePattern.MatchString(token):
			parsed.line = token
		case token == "master":
			parsed.line = "master"
		}
	}
	// BtbN daily assets are named ffmpeg-N-<hash>-<target>-<variant>.zip.
	if parsed.line == "" && strings.Contains(lower, "-n-") {
		parsed.line = "master"
	}
	if parsed.target == "" || parsed.variant == "" || parsed.line == "" {
		return parsedAsset{}, false
	}
	return parsed, true
}

func lineRank(line string) (int, int, bool) {
	if !stableLinePattern.MatchString(line) {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(line, "n"), ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor := 0
	if len(parts) > 1 {
		if minor, err = strconv.Atoi(parts[1]); err != nil {
			return 0, 0, false
		}
	}
	return major, minor, true
}

// selectAsset picks the single asset matching the query, applying the stable /
// daily line rules. It never guesses: an empty match is OMF-E3003.
func selectAsset(assets []github.Asset, query assetQuery) (github.Asset, string, error) {
	variant, linkage := normalizeVariant(query.Variant, query.Linkage)
	candidates := make([]parsedAsset, 0, len(assets))
	for _, asset := range assets {
		parsed, ok := parseAssetName(asset.Name)
		if !ok {
			continue
		}
		if parsed.target != query.Target || parsed.variant != variant || parsed.linkage != linkage {
			continue
		}
		parsed.asset = asset
		candidates = append(candidates, parsed)
	}
	if len(candidates) == 0 {
		return github.Asset{}, "", assetNotFound(query, variant, linkage)
	}

	// Prefer the documented archive format per target: Windows zip, Linux tar.xz.
	expectedExtension := ".zip"
	if query.Target == "linux64" {
		expectedExtension = ".tar.xz"
	}
	preferred := make([]parsedAsset, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.extension == expectedExtension {
			preferred = append(preferred, candidate)
		}
	}
	if len(preferred) > 0 {
		candidates = preferred
	}

	if query.ReleaseLine != "" {
		wanted := strings.ToLower(strings.TrimSpace(query.ReleaseLine))
		filtered := make([]parsedAsset, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.line == wanted {
				filtered = append(filtered, candidate)
			}
		}
		if len(filtered) == 0 {
			return github.Asset{}, "", assetNotFound(query, variant, linkage)
		}
		candidates = filtered
	} else if query.Channel == "stable" {
		stable := make([]parsedAsset, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.line != "master" {
				stable = append(stable, candidate)
			}
		}
		if len(stable) == 0 {
			return github.Asset{}, "", assetNotFound(query, variant, linkage)
		}
		candidates = stable
	} else {
		master := make([]parsedAsset, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.line == "master" {
				master = append(master, candidate)
			}
		}
		if len(master) == 0 {
			return github.Asset{}, "", apperrors.New(apperrors.E3003, "daily channel requires a master build").WithDetails(map[string]any{
				"channel": query.Channel,
				"target":  query.Target,
				"variant": variant,
			})
		}
		candidates = master
	}

	sort.SliceStable(candidates, func(left, right int) bool {
		leftMajor, leftMinor, leftStable := lineRank(candidates[left].line)
		rightMajor, rightMinor, rightStable := lineRank(candidates[right].line)
		switch {
		case leftStable && !rightStable:
			return true
		case !leftStable && rightStable:
			return false
		case leftStable && rightStable:
			if leftMajor != rightMajor {
				return leftMajor > rightMajor
			}
			if leftMinor != rightMinor {
				return leftMinor > rightMinor
			}
		}
		return candidates[left].asset.Name < candidates[right].asset.Name
	})
	if len(candidates) > 1 {
		first, second := candidates[0], candidates[1]
		if first.line == second.line && first.asset.Name != second.asset.Name {
			return github.Asset{}, "", apperrors.New(apperrors.E3003, "multiple assets match the same release line").WithDetails(map[string]any{
				"channel": query.Channel,
				"target":  query.Target,
				"variant": variant,
				"line":    first.line,
				"assets":  []string{first.asset.Name, second.asset.Name},
			})
		}
	}
	selected := candidates[0]
	return selected.asset, selected.line, nil
}

func assetNotFound(query assetQuery, variant, linkage string) error {
	return apperrors.New(apperrors.E3003, "no asset matches the requested target/variant/linkage").WithDetails(map[string]any{
		"channel":      query.Channel,
		"target":       query.Target,
		"variant":      variant,
		"linkage":      linkage,
		"release_line": query.ReleaseLine,
	})
}
