package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func CreateBackup(filePath string) (string, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filePath + ".backup." + timestamp

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("write backup: %w", err)
	}

	CleanupOldBackups(filePath, 3)

	return backupPath, nil
}

func RestoreBackup(filePath string, backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("restore file: %w", err)
	}

	return nil
}

func CleanupOldBackups(filePath string, maxCount int) {
	dir := filePath[:strings.LastIndex(filePath, "/")]
	base := filePath[strings.LastIndex(filePath, "/")+1:]

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	backups := []string{}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), base+".backup.") {
			backups = append(backups, filepath.Join(dir, f.Name()))
		}
	}

	sort.Strings(backups)

	for i := 0; i < len(backups)-maxCount; i++ {
		os.Remove(backups[i])
	}

	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > 7*24*time.Hour {
			os.Remove(backup)
		}
	}
}

func GetMostRecentBackup(filePath string) string {
	dir := filePath[:strings.LastIndex(filePath, "/")]
	base := filePath[strings.LastIndex(filePath, "/")+1:]

	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	backups := []string{}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), base+".backup.") {
			backups = append(backups, filepath.Join(dir, f.Name()))
		}
	}

	if len(backups) == 0 {
		return ""
	}

	sort.Strings(backups)
	return backups[len(backups)-1]
}
