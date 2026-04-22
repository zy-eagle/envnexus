package marketplace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
)

const (
	ideExtensionItemName    = "EnvNexus IDE Sync"
	ideExtensionVersion     = "0.1.0"
	ideExtensionAuthor      = "EnvNexus"
	ideExtensionDescription = "Sync skills and rules with EnvNexus from your IDE (VSIX marketplace plugin)."
)

// SeedIDEExtension upserts the bundled Visual Studio Code extension (.vsix) as a published plugin marketplace item.
// If the file is missing, object storage is not configured, or the path is empty, it logs and returns nil.
func SeedIDEExtension(ctx context.Context, svc *Service, vsixPath string) error {
	if svc == nil {
		return nil
	}
	if svc.minioClient == nil {
		slog.Info("marketplace: skipping IDE extension seed (object storage not configured)")
		return nil
	}
	vsixPath = strings.TrimSpace(vsixPath)
	if vsixPath == "" {
		return nil
	}
	fi, err := os.Stat(vsixPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("marketplace: IDE extension .vsix not found, skipping seed", "path", vsixPath)
			return nil
		}
		return err
	}
	if fi.IsDir() {
		slog.Info("marketplace: IDE extension path is a directory, skipping seed", "path", vsixPath)
		return nil
	}
	if fi.Size() == 0 {
		slog.Info("marketplace: IDE extension .vsix is empty, skipping seed", "path", vsixPath)
		return nil
	}
	if fi.Size() > maxMarketplaceFileBytes {
		return fmt.Errorf("IDE extension .vsix exceeds max size: %d bytes", maxMarketplaceFileBytes)
	}

	f, err := os.Open(vsixPath)
	if err != nil {
		return err
	}
	defer f.Close()

	pluginType := domain.MarketplaceItemTypePlugin
	items, _, err := svc.repo.ListMarketplaceItems(ctx, &pluginType, nil, 1, 2000)
	if err != nil {
		return err
	}
	var existing *domain.MarketplaceItem
	for _, it := range items {
		if it != nil && it.Name == ideExtensionItemName {
			existing = it
			break
		}
	}

	filename := filepath.Base(vsixPath)
	if filename == "" || filename == "." {
		filename = "envnexus-sync-0.1.0.vsix"
	}

	if existing == nil {
		_, err := svc.CreateMarketplaceItem(ctx, CreateMarketplaceItemInput{
			Type:        domain.MarketplaceItemTypePlugin,
			Name:        ideExtensionItemName,
			Description: ideExtensionDescription,
			Version:     ideExtensionVersion,
			Author:      ideExtensionAuthor,
			Status:      domain.MarketplaceItemStatusPublished,
			File:        f,
			FileSize:    fi.Size(),
			Filename:    filename,
			ContentType: "application/octet-stream",
		})
		if err != nil {
			return err
		}
		slog.Info("marketplace: seeded IDE extension plugin", "name", ideExtensionItemName, "version", ideExtensionVersion, "file", filename)
		return nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	_, err = svc.UpdateMarketplaceItem(ctx, existing.ID, UpdateMarketplaceItemInput{
		Name:        ideExtensionItemName,
		Description: ideExtensionDescription,
		Version:     ideExtensionVersion,
		Author:      ideExtensionAuthor,
		Status:      string(domain.MarketplaceItemStatusPublished),
		File:        f,
		FileSize:    fi.Size(),
		Filename:    filename,
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return err
	}
	slog.Info("marketplace: updated IDE extension plugin", "id", existing.ID, "name", ideExtensionItemName, "version", ideExtensionVersion, "file", filename)
	return nil
}
