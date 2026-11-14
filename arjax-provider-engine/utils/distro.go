package utils

import (
	"bufio"
	"os"
	"strings"
)

// DistroInfo contains distribution metadata
type DistroInfo struct {
	ID        string
	IDLike    string
	Name      string
	VersionID string
}

// DetectDistro reads /etc/os-release to identify distribution
func DetectDistro() DistroInfo {
	info := DistroInfo{}
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.Trim(parts[1], `"`)

		switch key {
		case "ID":
			info.ID = val
		case "ID_LIKE":
			info.IDLike = val
		case "NAME":
			info.Name = val
		case "VERSION_ID":
			info.VersionID = val
		}
	}
	return info
}

// DetectPackageManager determines standard package manager
func DetectPackageManager() string {
	distro := DetectDistro()
	id := strings.ToLower(distro.ID)
	idLike := strings.ToLower(distro.IDLike)

	if CommandExists("paru") && (id == "arch" || strings.Contains(idLike, "arch") || id == "manjaro" || id == "endeavouros") {
		return "paru"
	}
	if CommandExists("pacman") && (id == "arch" || strings.Contains(idLike, "arch") || id == "manjaro" || id == "endeavouros") {
		return "pacman"
	}
	if CommandExists("apt") && (id == "ubuntu" || id == "debian" || strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") || id == "pop" || id == "mint") {
		return "apt"
	}
	if CommandExists("dnf") && (id == "fedora" || id == "rhel" || id == "centos" || strings.Contains(idLike, "fedora")) {
		return "dnf"
	}
	if CommandExists("zypper") && (id == "opensuse" || id == "sles" || strings.Contains(idLike, "suse")) {
		return "zypper"
	}

	// Fallback based on binary presence
	if CommandExists("paru") {
		return "paru"
	}
	if CommandExists("pacman") {
		return "pacman"
	}
	if CommandExists("apt") {
		return "apt"
	}
	if CommandExists("dnf") {
		return "dnf"
	}
	if CommandExists("zypper") {
		return "zypper"
	}
	return ""
}
