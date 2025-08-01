package main

import (
	"context"
	"encoding/json"
	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v55/github"
	"os"
	"path/filepath"
	"strings"
)

// ReleaseAsset represents an asset grouped by version
type ReleaseAsset struct {
	Version string
	Name    string
	URL     string
}

// FetchReleaseAssets fetches releases and parses assets
func FetchReleaseAssets(ctx context.Context, owner, repo string) ([]ReleaseAsset, error) {
	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, nil)
	if err != nil {
		return nil, err
	}
	var assets []ReleaseAsset
	versionAssets := make(map[string][]ReleaseAsset)
	for _, rel := range releases {
		version := rel.GetTagName()
		// Exclude RC and beta releases
		if strings.Contains(strings.ToLower(version), "rc") || strings.Contains(strings.ToLower(version), "beta") {
			continue
		}
		if _, err := semver.NewVersion(version); err != nil {
			continue // skip invalid semver
		}
		for _, asset := range rel.Assets {
			name := asset.GetName()
			versionAssets[version] = append(versionAssets[version], ReleaseAsset{
				Version: version,
				Name:    name,
				URL:     asset.GetBrowserDownloadURL(),
			})
		}
	}
	for _, assetList := range versionAssets {
		assets = append(assets, assetList...)
	}
	return assets, nil
}

// GetCachedReleaseAssets returns cached assets if available, otherwise fetches and caches them
func GetCachedReleaseAssets(ctx context.Context, owner, repo string) ([]ReleaseAsset, error) {
	cacheFile := filepath.Join(os.TempDir(), "kairos_releases_cache.json")
	if data, err := os.ReadFile(cacheFile); err == nil {
		var assets []ReleaseAsset
		if err := json.Unmarshal(data, &assets); err == nil {
			return assets, nil
		}
	}
	assets, err := FetchReleaseAssets(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(assets)
	if err == nil {
		_ = os.WriteFile(cacheFile, data, 0644)
	}
	return assets, nil
}
