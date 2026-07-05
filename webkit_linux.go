//go:build linux

package main

import "os"

// Na Linuxu (WebKitGTK) se často stává, že se okno otevře prázdné/bílé kvůli
// chybě v DMABUF rendereru nebo GPU compositingu (typicky ve VM, na GNOME,
// se staršími/virtuálními GPU). Nastavením těchto proměnných PŘED startem
// webview to obejdeme, aby uživatel nemusel nic ručně nastavovat.
//
// init() běží před main(), takže proměnné jsou nastavené včas.
func init() {
	// Vypni DMABUF renderer (nejčastější příčina bílého okna)
	if _, ok := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); !ok {
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	// Vypni HW compositing (pomáhá ve VM / bez GPU akcelerace)
	if _, ok := os.LookupEnv("WEBKIT_DISABLE_COMPOSITING_MODE"); !ok {
		os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
}
