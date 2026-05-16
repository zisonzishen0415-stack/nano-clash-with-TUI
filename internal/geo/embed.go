package geo

import (
	"embed"
	"io"
	"os"
	"path/filepath"
)

//go:embed Country.mmdb geoip.dat
var geoFiles embed.FS

func ExtractMMDB(dest string) error {
	f, err := geoFiles.Open("Country.mmdb")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, f)
	return err
}

func ExtractGeosite(dest string) error {
	f, err := geoFiles.Open("geoip.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, f)
	return err
}

func EnsureGeoData(baseDir string) error {
	mmdb := filepath.Join(baseDir, "Country.mmdb")
	geosite := filepath.Join(baseDir, "geosite.dat")

	if _, err := os.Stat(mmdb); os.IsNotExist(err) {
		if err := ExtractMMDB(mmdb); err != nil {
			return err
		}
	}

	if _, err := os.Stat(geosite); os.IsNotExist(err) {
		if err := ExtractGeosite(geosite); err != nil {
			return err
		}
	}

	return nil
}
