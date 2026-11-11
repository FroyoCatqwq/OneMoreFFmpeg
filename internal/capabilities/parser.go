package capabilities

import (
	"bufio"
	"regexp"
	"sort"
	"strings"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var (
	flagPattern    = regexp.MustCompile(`^[A-Za-z.]{1,8}$`)
	versionPattern = regexp.MustCompile(`(?im)^(?:ffmpeg|ffprobe|ffplay)\s+version\s+(\S+)`)
)

func parseVersion(raw string) model.VersionInfo {
	info := model.VersionInfo{Raw: raw}
	if match := versionPattern.FindStringSubmatch(raw); len(match) >= 2 {
		info.Version = match[1]
	}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "configuration:") {
			info.Configuration = strings.TrimSpace(trimmed[len("configuration:"):])
			break
		}
	}
	return info
}

func parseLicense(primary, fallback string) model.LicenseInfo {
	raw := primary
	if strings.TrimSpace(raw) == "" {
		raw = fallback
	}
	info := model.LicenseInfo{Raw: raw}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			info.Summary = trimmed
			break
		}
	}
	return info
}

func parseList(kind, raw string) ([]model.CapabilityEntry, int) {
	switch kind {
	case "muxers", "demuxers", "devices", "decoders", "encoders", "filters", "pixel_formats":
		return parseFlagList(raw)
	case "channel_layouts", "sample_formats":
		return parseSimpleList(raw)
	default:
		return []model.CapabilityEntry{}, 0
	}
}

func parseFlagList(raw string) ([]model.CapabilityEntry, int) {
	entries := make([]model.CapabilityEntry, 0)
	unknown := 0
	scanner := newScanner(raw)
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if skipCapabilityLine(line) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			unknown++
			continue
		}
		flags := fields[0]
		if flagPattern.MatchString(flags) {
			name := fields[1]
			description := strings.TrimSpace(strings.TrimPrefix(line, flags+" "+name))
			entries = append(entries, model.CapabilityEntry{
				Name:        name,
				Flags:       flags,
				Description: description,
				Raw:         rawLine,
			})
			continue
		}
		name := fields[0]
		description := strings.TrimSpace(strings.TrimPrefix(line, name))
		entries = append(entries, model.CapabilityEntry{
			Name:        name,
			Description: description,
			Raw:         rawLine,
		})
	}
	return normalizeEntries(entries), unknown
}

func parseSimpleList(raw string) ([]model.CapabilityEntry, int) {
	entries := make([]model.CapabilityEntry, 0)
	unknown := 0
	scanner := newScanner(raw)
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if skipCapabilityLine(line) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			unknown++
			continue
		}
		name := fields[0]
		description := strings.TrimSpace(strings.TrimPrefix(line, name))
		entries = append(entries, model.CapabilityEntry{
			Name:        name,
			Description: description,
			Raw:         rawLine,
		})
	}
	return normalizeEntries(entries), unknown
}

func newScanner(raw string) *bufio.Scanner {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	return scanner
}

func skipCapabilityLine(line string) bool {
	if line == "" {
		return true
	}
	lower := strings.ToLower(line)
	if strings.HasSuffix(line, ":") {
		return true
	}
	if strings.HasPrefix(line, "--") {
		return true
	}
	if strings.Contains(line, " = ") {
		return true
	}
	if strings.HasPrefix(lower, "configuration:") {
		return true
	}
	if lower == "name depth" || lower == "name description" || lower == "name decomposition" {
		return true
	}
	if strings.HasPrefix(lower, "name ") && (strings.Contains(lower, "description") || strings.Contains(lower, "depth") || strings.Contains(lower, "decomposition")) {
		return true
	}
	if lower == "input:" || lower == "output:" || lower == "supported:" {
		return true
	}
	return false
}

func normalizeEntries(entries []model.CapabilityEntry) []model.CapabilityEntry {
	seen := map[string]bool{}
	out := make([]model.CapabilityEntry, 0, len(entries))
	for _, entry := range entries {
		key := entry.Name + "\x00" + entry.Flags + "\x00" + entry.Description
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Flags != out[j].Flags {
			return out[i].Flags < out[j].Flags
		}
		return out[i].Description < out[j].Description
	})
	return out
}
