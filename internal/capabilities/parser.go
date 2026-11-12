package capabilities

import (
	"bufio"
	"regexp"
	"sort"
	"strings"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var (
	flagPattern     = regexp.MustCompile(`^[A-Za-z.]{1,8}$`)
	versionPattern  = regexp.MustCompile(`(?im)^(?:ffmpeg|ffprobe|ffplay)\s+version\s+(\S+)`)
	helpMissPattern = regexp.MustCompile(`^(?:Codec '[^']+' is not recognized by FFmpeg\.|Unknown (?:filter|format|decoder|encoder) '[^']+'\.)`)
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
		if trimmed == "" || isBannerLine(trimmed) {
			continue
		}
		info.Summary = trimmed
		break
	}
	return info
}

// isBannerLine keeps a leaked build banner from being reported as the license.
func isBannerLine(line string) bool {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"ffmpeg version ", "ffprobe version ", "ffplay version ", "built with ", "configuration:", "libav", "libsw", "libpostproc"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// helpTopicMissing reports whether FFmpeg answered a help topic with a
// "not found" message instead of help text. FFmpeg exits 0 in that case and
// has written the diagnostic to stdout or stderr depending on version.
func helpTopicMissing(stdout, stderr string) bool {
	if firstContentLineMatches(stdout) || firstContentLineMatches(stderr) {
		return true
	}
	// A successful help command always prints something; empty stdout means
	// the topic produced no help at all.
	return strings.TrimSpace(stdout) == ""
}

func firstContentLineMatches(raw string) bool {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return helpMissPattern.MatchString(trimmed)
	}
	return false
}

func parseList(kind, raw string) ([]model.CapabilityEntry, int) {
	switch kind {
	case "muxers", "demuxers", "devices", "decoders", "encoders", "filters", "pixel_formats":
		return parseFlagList(raw)
	case "channel_layouts":
		return parseChannelLayouts(raw)
	case "sample_formats":
		return parseSimpleList(raw)
	default:
		return []model.CapabilityEntry{}, 0
	}
}

// parseChannelLayouts keeps only the "Standard channel layouts" section.
// FFmpeg prints individual channels and standard layouts in one output; mixing
// them would report FL/FR as if they were standalone layouts.
func parseChannelLayouts(raw string) ([]model.CapabilityEntry, int) {
	lines := strings.Split(raw, "\n")
	for index, line := range lines {
		if strings.EqualFold(strings.Join(strings.Fields(line), " "), "standard channel layouts:") {
			return parseSimpleList(strings.Join(lines[index+1:], "\n"))
		}
	}
	return parseSimpleList(raw)
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
		if isColumnHeader(fields) {
			continue
		}
		flags := fields[0]
		if flagPattern.MatchString(flags) {
			entries = append(entries, model.CapabilityEntry{
				Name:        fields[1],
				Flags:       flags,
				Description: strings.Join(fields[2:], " "),
				Raw:         rawLine,
			})
			continue
		}
		entries = append(entries, model.CapabilityEntry{
			Name:        fields[0],
			Description: strings.Join(fields[1:], " "),
			Raw:         rawLine,
		})
	}
	if scanner.Err() != nil {
		unknown++
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
		entries = append(entries, model.CapabilityEntry{
			Name:        fields[0],
			Description: strings.Join(fields[1:], " "),
			Raw:         rawLine,
		})
	}
	if scanner.Err() != nil {
		unknown++
	}
	return normalizeEntries(entries), unknown
}

// isColumnHeader matches table headers such as
// "FLAGS NAME            NB_COMPONENTS BITS_PER_PIXEL BIT_DEPTHS".
// FFmpeg prints them to the same stream as data rows.
func isColumnHeader(fields []string) bool {
	return len(fields) >= 2 && strings.EqualFold(fields[0], "flags") && strings.EqualFold(fields[1], "name")
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
