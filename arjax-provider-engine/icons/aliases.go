package icons

import (
	"strings"
	"sync"
)

// AppAliasEntry links a canonical reverse-DNS application ID with distribution package names
type AppAliasEntry struct {
	CanonicalID string
	Name        string
	Category    string
	Aliases     []string
}

// AliasRegistry manages bidirectional mapping between canonical AppIDs and distro package names
type AliasRegistry struct {
	mu           sync.RWMutex
	canonicalMap map[string]*AppAliasEntry // canonicalID -> entry
	aliasToCanon map[string]string         // alias -> canonicalID
}

var defaultRegistry *AliasRegistry
var onceInitRegistry sync.Once

// GetAliasRegistry returns the singleton registry with pre-populated common applications
func GetAliasRegistry() *AliasRegistry {
	onceInitRegistry.Do(func() {
		defaultRegistry = NewAliasRegistry()
		defaultRegistry.populateStandardAliases()
	})
	return defaultRegistry
}

func NewAliasRegistry() *AliasRegistry {
	return &AliasRegistry{
		canonicalMap: make(map[string]*AppAliasEntry),
		aliasToCanon: make(map[string]string),
	}
}

func (r *AliasRegistry) Register(canonicalID, name, category string, aliases ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cleanID := strings.TrimSpace(canonicalID)
	if cleanID == "" {
		return
	}

	normID := strings.ToLower(cleanID)
	cleanAliases := make([]string, 0, len(aliases)+2)
	cleanAliases = append(cleanAliases, normID)

	// If ID has parts, e.g. "org.mozilla.firefox", add the leaf name "firefox"
	parts := strings.Split(normID, ".")
	if len(parts) > 1 {
		cleanAliases = append(cleanAliases, parts[len(parts)-1])
	}

	for _, a := range aliases {
		cleanA := strings.ToLower(strings.TrimSpace(a))
		if cleanA != "" {
			cleanAliases = append(cleanAliases, cleanA)
		}
	}

	entry := &AppAliasEntry{
		CanonicalID: cleanID,
		Name:        name,
		Category:    category,
		Aliases:     cleanAliases,
	}

	r.canonicalMap[normID] = entry
	for _, a := range cleanAliases {
		r.aliasToCanon[a] = cleanID
	}
}

// ResolveCanonical returns the canonical reverse-DNS AppID if known, or the cleanest input ID
func (r *AliasRegistry) ResolveCanonical(input string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	norm := strings.ToLower(strings.TrimSpace(input))
	if norm == "" {
		return ""
	}

	if canon, exists := r.aliasToCanon[norm]; exists {
		return canon
	}

	// Try stripping common package suffixes: -bin, -git, -desktop, -app
	trimmed := norm
	for _, sfx := range []string{"-bin", "-git", "-desktop", "-app", "-electron", "_bin"} {
		if strings.HasSuffix(trimmed, sfx) {
			trimmed = strings.TrimSuffix(trimmed, sfx)
			if canon, exists := r.aliasToCanon[trimmed]; exists {
				return canon
			}
		}
	}

	// If already has dots, assume it's already a reverse-DNS canonical ID
	if strings.Contains(input, ".") {
		return input
	}

	return input
}

// GetAliases returns all known aliases and package names for a given application ID
func (r *AliasRegistry) GetAliases(appID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canon := r.ResolveCanonical(appID)
	if entry, exists := r.canonicalMap[strings.ToLower(canon)]; exists {
		return append([]string(nil), entry.Aliases...)
	}

	norm := strings.ToLower(strings.TrimSpace(appID))
	parts := strings.Split(norm, ".")
	if len(parts) > 1 {
		return []string{norm, parts[len(parts)-1]}
	}
	return []string{norm}
}

func (r *AliasRegistry) populateStandardAliases() {
	apps := []struct {
		id       string
		name     string
		cat      string
		aliases  []string
	}{
		{"com.brave.Browser", "Brave Browser", "Browser", []string{"brave", "brave-bin", "brave-browser"}},
		{"com.spotify.Client", "Spotify", "Audio", []string{"spotify", "spotify-launcher"}},
		{"com.discordapp.Discord", "Discord", "Chat", []string{"discord", "discord-canary", "discord-ptb"}},
		{"org.mozilla.firefox", "Firefox", "Browser", []string{"firefox", "firefox-esr", "firefox-developer-edition"}},
		{"com.valvesoftware.Steam", "Steam", "Game", []string{"steam", "steam-native", "steam-runtime"}},
		{"org.blender.Blender", "Blender", "Graphics", []string{"blender"}},
		{"org.videolan.VLC", "VLC media player", "AudioVideo", []string{"vlc"}},
		{"com.visualstudio.code", "Visual Studio Code", "Development", []string{"vscode", "visual-studio-code-bin", "code", "code-oss"}},
		{"org.gimp.GIMP", "GNU Image Manipulation Program", "Graphics", []string{"gimp", "gimp-2.99"}},
		{"org.telegram.desktop", "Telegram Desktop", "Chat", []string{"telegram", "telegram-desktop", "telegramdesktop"}},
		{"com.bitwarden.desktop", "Bitwarden", "Utility", []string{"bitwarden", "bitwarden-bin"}},
		{"com.obsproject.Studio", "OBS Studio", "AudioVideo", []string{"obs", "obs-studio"}},
		{"org.kde.kdenlive", "Kdenlive", "AudioVideo", []string{"kdenlive"}},
		{"org.audacityteam.Audacity", "Audacity", "Audio", []string{"audacity"}},
		{"org.inkscape.Inkscape", "Inkscape", "Graphics", []string{"inkscape"}},
		{"com.heroicgameslauncher.hgl", "Heroic Games Launcher", "Game", []string{"heroic", "heroic-games-launcher-bin"}},
		{"net.lutris.Lutris", "Lutris", "Game", []string{"lutris"}},
		{"org.prismlauncher.PrismLauncher", "Prism Launcher", "Game", []string{"prismlauncher", "prism-launcher"}},
		{"org.mozilla.Thunderbird", "Thunderbird", "Network", []string{"thunderbird"}},
		{"com.google.Chrome", "Google Chrome", "Browser", []string{"google-chrome", "google-chrome-stable", "chrome"}},
		{"com.slack.Slack", "Slack", "Chat", []string{"slack", "slack-desktop"}},
		{"io.neovim.nvim", "Neovim", "Development", []string{"neovim", "nvim"}},
		{"org.kde.krita", "Krita", "Graphics", []string{"krita"}},
		{"org.godotengine.Godot", "Godot Engine", "Development", []string{"godot", "godot-mono"}},
		{"com.getpostman.Postman", "Postman", "Development", []string{"postman", "postman-bin"}},
		{"io.dbeaver.DBeaverCommunity", "DBeaver Community", "Development", []string{"dbeaver", "dbeaver-ce"}},
		{"io.mpv.Mpv", "mpv Media Player", "AudioVideo", []string{"mpv"}},
		{"com.calibre_ebook.calibre", "Calibre", "Office", []string{"calibre"}},
		{"org.qbittorrent.qBittorrent", "qBittorrent", "Network", []string{"qbittorrent"}},
		{"fr.handbrake.ghb", "HandBrake", "AudioVideo", []string{"handbrake", "ghb"}},
		{"org.libreoffice.LibreOffice", "LibreOffice", "Office", []string{"libreoffice", "libreoffice-fresh", "libreoffice-still"}},
		{"org.wireshark.Wireshark", "Wireshark", "Network", []string{"wireshark", "wireshark-qt"}},
		{"rest.insomnia.Insomnia", "Insomnia", "Development", []string{"insomnia"}},
		{"org.libretro.RetroArch", "RetroArch", "Game", []string{"retroarch"}},
		{"com.usebottles.bottles", "Bottles", "Utility", []string{"bottles"}},
		{"org.onlyoffice.desktopeditors", "OnlyOffice", "Office", []string{"onlyoffice", "onlyoffice-bin"}},
	}

	for _, a := range apps {
		r.Register(a.id, a.name, a.cat, a.aliases...)
	}
}

