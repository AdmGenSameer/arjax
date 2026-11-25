package identity

import "sync"

var (
	defaultIdentityRegistry *Registry
	onceInitIdentity        sync.Once
)

// GetDefaultRegistry returns the shared singleton application identity registry.
func GetDefaultRegistry() *Registry {
	onceInitIdentity.Do(func() {
		defaultIdentityRegistry = NewRegistry()
		populateStandardIdentities(defaultIdentityRegistry)
	})
	return defaultIdentityRegistry
}

func populateStandardIdentities(r *Registry) {
	entries := []AppIdentityEntry{
		{
			CanonicalID:    "com.brave.Browser",
			Name:           "Brave Browser",
			Category:       "Browser",
			PackageAliases: []string{"brave", "brave-bin", "brave-browser"},
			ProviderHints:  map[string]string{"aur": "brave-bin", "flatpak": "com.brave.Browser"},
			IconNames:      []string{"brave-browser", "brave-desktop", "brave"},
		},
		{
			CanonicalID:    "com.spotify.Client",
			Name:           "Spotify",
			Category:       "Audio",
			PackageAliases: []string{"spotify", "spotify-launcher"},
			ProviderHints:  map[string]string{"aur": "spotify", "pacman": "spotify-launcher", "flatpak": "com.spotify.Client"},
			IconNames:      []string{"spotify-client", "spotify"},
		},
		{
			CanonicalID:    "com.discordapp.Discord",
			Name:           "Discord",
			Category:       "Chat",
			PackageAliases: []string{"discord"},
			ProviderHints:  map[string]string{"pacman": "discord", "flatpak": "com.discordapp.Discord"},
			IconNames:      []string{"discord", "com.discordapp.Discord"},
		},
		{
			CanonicalID:    "org.mozilla.firefox",
			Name:           "Firefox",
			Category:       "Browser",
			PackageAliases: []string{"firefox"},
			ProviderHints:  map[string]string{"pacman": "firefox", "aur": "firefox-bin", "flatpak": "org.mozilla.firefox"},
			IconNames:      []string{"firefox", "firefox-esr", "firefox-icon"},
		},
		{
			CanonicalID:    "com.valvesoftware.Steam",
			Name:           "Steam",
			Category:       "Game",
			PackageAliases: []string{"steam", "steam-native-runtime", "steam-native"},
			ProviderHints:  map[string]string{"pacman": "steam", "flatpak": "com.valvesoftware.Steam"},
			IconNames:      []string{"steam", "steam_tray_mono"},
		},
		{
			CanonicalID:    "org.blender.Blender",
			Name:           "Blender",
			Category:       "Graphics",
			PackageAliases: []string{"blender"},
			IconNames:      []string{"blender"},
		},
		{
			CanonicalID:    "org.videolan.VLC",
			Name:           "VLC media player",
			Category:       "AudioVideo",
			PackageAliases: []string{"vlc"},
			IconNames:      []string{"vlc"},
		},
		{
			CanonicalID:    "com.visualstudio.code",
			Name:           "Visual Studio Code",
			Category:       "Development",
			PackageAliases: []string{"vscode", "visual-studio-code-bin", "code"},
			ProviderHints:  map[string]string{"pacman": "code", "aur": "visual-studio-code-bin", "flatpak": "com.visualstudio.code"},
			IconNames:      []string{"visual-studio-code", "vscode", "code-oss", "com.visualstudio.code"},
		},
		{
			CanonicalID:    "org.gimp.GIMP",
			Name:           "GNU Image Manipulation Program",
			Category:       "Graphics",
			PackageAliases: []string{"gimp"},
			IconNames:      []string{"gimp"},
		},
		{
			CanonicalID:    "org.telegram.desktop",
			Name:           "Telegram Desktop",
			Category:       "Chat",
			PackageAliases: []string{"telegram-desktop", "telegram-desktop-bin", "telegram"},
			IconNames:      []string{"telegram", "telegram-desktop"},
		},
		{
			CanonicalID:    "com.bitwarden.desktop",
			Name:           "Bitwarden",
			Category:       "Utility",
			PackageAliases: []string{"bitwarden", "bitwarden-bin"},
			IconNames:      []string{"bitwarden"},
		},
		{
			CanonicalID:    "com.obsproject.Studio",
			Name:           "OBS Studio",
			Category:       "AudioVideo",
			PackageAliases: []string{"obs-studio", "obs"},
			IconNames:      []string{"com.obsproject.Studio", "obs"},
		},
		{
			CanonicalID:    "org.kde.kdenlive",
			Name:           "Kdenlive",
			Category:       "AudioVideo",
			PackageAliases: []string{"kdenlive"},
			IconNames:      []string{"kdenlive"},
		},
		{
			CanonicalID:    "org.audacityteam.Audacity",
			Name:           "Audacity",
			Category:       "Audio",
			PackageAliases: []string{"audacity"},
			IconNames:      []string{"audacity"},
		},
		{
			CanonicalID:    "org.inkscape.Inkscape",
			Name:           "Inkscape",
			Category:       "Graphics",
			PackageAliases: []string{"inkscape"},
			IconNames:      []string{"inkscape", "org.inkscape.Inkscape"},
		},
		{
			CanonicalID:    "com.heroicgameslauncher.hgl",
			Name:           "Heroic Games Launcher",
			Category:       "Game",
			PackageAliases: []string{"heroic", "heroic-games-launcher-bin"},
			IconNames:      []string{"heroic", "com.heroicgameslauncher.hgl"},
		},
		{
			CanonicalID:    "net.lutris.Lutris",
			Name:           "Lutris",
			Category:       "Game",
			PackageAliases: []string{"lutris"},
			IconNames:      []string{"lutris"},
		},
		{
			CanonicalID:    "org.prismlauncher.PrismLauncher",
			Name:           "Prism Launcher",
			Category:       "Game",
			PackageAliases: []string{"prismlauncher", "prism-launcher"},
			IconNames:      []string{"prismlauncher", "org.prismlauncher.PrismLauncher"},
		},
		{
			CanonicalID:    "org.mozilla.Thunderbird",
			Name:           "Thunderbird",
			Category:       "Network",
			PackageAliases: []string{"thunderbird"},
			IconNames:      []string{"thunderbird"},
		},
		{
			CanonicalID:    "com.google.Chrome",
			Name:           "Google Chrome",
			Category:       "Browser",
			PackageAliases: []string{"google-chrome", "google-chrome-stable"},
			IconNames:      []string{"google-chrome"},
		},
		{
			CanonicalID:    "com.slack.Slack",
			Name:           "Slack",
			Category:       "Chat",
			PackageAliases: []string{"slack-desktop", "slack"},
			IconNames:      []string{"slack"},
		},
		{
			CanonicalID:    "io.neovim.nvim",
			Name:           "Neovim",
			Category:       "Development",
			PackageAliases: []string{"neovim", "nvim"},
			IconNames:      []string{"nvim", "neovim"},
		},
		{
			CanonicalID:    "org.kde.krita",
			Name:           "Krita",
			Category:       "Graphics",
			PackageAliases: []string{"krita"},
			IconNames:      []string{"krita"},
		},
		{
			CanonicalID:    "org.godotengine.Godot",
			Name:           "Godot Engine",
			Category:       "Development",
			PackageAliases: []string{"godot", "godot-mono"},
			IconNames:      []string{"godot"},
		},
		{
			CanonicalID:    "com.getpostman.Postman",
			Name:           "Postman",
			Category:       "Development",
			PackageAliases: []string{"postman", "postman-bin"},
			IconNames:      []string{"postman"},
		},
		{
			CanonicalID:    "io.dbeaver.DBeaverCommunity",
			Name:           "DBeaver Community",
			Category:       "Development",
			PackageAliases: []string{"dbeaver", "dbeaver-ce"},
			IconNames:      []string{"dbeaver"},
		},
		{
			CanonicalID:    "io.mpv.Mpv",
			Name:           "mpv Media Player",
			Category:       "AudioVideo",
			PackageAliases: []string{"mpv"},
			IconNames:      []string{"mpv"},
		},
		{
			CanonicalID:    "com.calibre_ebook.calibre",
			Name:           "Calibre",
			Category:       "Office",
			PackageAliases: []string{"calibre"},
			IconNames:      []string{"calibre-gui", "calibre"},
		},
		{
			CanonicalID:    "org.qbittorrent.qBittorrent",
			Name:           "qBittorrent",
			Category:       "Network",
			PackageAliases: []string{"qbittorrent"},
			IconNames:      []string{"qbittorrent"},
		},
		{
			CanonicalID:    "fr.handbrake.ghb",
			Name:           "HandBrake",
			Category:       "AudioVideo",
			PackageAliases: []string{"handbrake", "ghb"},
			IconNames:      []string{"fr.handbrake.ghb", "handbrake"},
		},
		{
			CanonicalID:    "org.libreoffice.LibreOffice",
			Name:           "LibreOffice",
			Category:       "Office",
			PackageAliases: []string{"libreoffice", "libreoffice-fresh", "libreoffice-still"},
			IconNames:      []string{"libreoffice-startcenter", "libreoffice"},
		},
		{
			CanonicalID:    "org.wireshark.Wireshark",
			Name:           "Wireshark",
			Category:       "Network",
			PackageAliases: []string{"wireshark-qt", "wireshark"},
			IconNames:      []string{"wireshark", "org.wireshark.Wireshark"},
		},
		{
			CanonicalID:    "rest.insomnia.Insomnia",
			Name:           "Insomnia",
			Category:       "Development",
			PackageAliases: []string{"insomnia", "insomnia-bin"},
			IconNames:      []string{"insomnia"},
		},
		{
			CanonicalID:    "org.libretro.RetroArch",
			Name:           "RetroArch",
			Category:       "Game",
			PackageAliases: []string{"retroarch"},
			IconNames:      []string{"retroarch"},
		},
		{
			CanonicalID:    "com.usebottles.bottles",
			Name:           "Bottles",
			Category:       "Utility",
			PackageAliases: []string{"bottles"},
			IconNames:      []string{"com.usebottles.bottles", "bottles"},
		},
		{
			CanonicalID:    "org.onlyoffice.desktopeditors",
			Name:           "OnlyOffice",
			Category:       "Office",
			PackageAliases: []string{"onlyoffice-bin", "onlyoffice"},
			IconNames:      []string{"onlyoffice-desktopeditors", "onlyoffice"},
		},
		{
			CanonicalID:    "md.obsidian.Obsidian",
			Name:           "Obsidian",
			Category:       "Office",
			PackageAliases: []string{"obsidian"},
			IconNames:      []string{"obsidian"},
		},
	}

	for _, e := range entries {
		r.Register(e)
	}
}

