package main

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var filteredAssets []ReleaseAsset // Store filtered assets for dropdowns
var saveFilePath string           // Store the path to save the downloaded file
// This covers the

func getDownloadWindow() *gtk.Button {
	downloadBtn := gtk.NewButtonWithLabel("Download ISOs")
	downloadBtn.ConnectClicked(func() {
		// Open a new window for ISO downloads
		downloadWin := gtk.NewWindow()
		downloadWin.SetTitle("Download ISO")
		downloadWin.SetDefaultSize(800, 600)

		// Create a vertical box for content
		vbox := gtk.NewBox(gtk.OrientationVertical, 10)
		vbox.SetMarginTop(20)
		vbox.SetMarginBottom(20)
		vbox.SetMarginStart(20)
		vbox.SetMarginEnd(20)
		vbox.SetHAlign(gtk.AlignFill)
		vbox.SetVAlign(gtk.AlignFill)
		vbox.SetHExpand(true)
		vbox.SetVExpand(true)
		vbox.SetSizeRequest(-1, -1) // Let vbox expand to fill window

		// Add a loading label and spinner
		loadingLabel := gtk.NewLabel("Loading data...")
		loadingLabel.SetHAlign(gtk.AlignCenter)
		spinner := gtk.NewSpinner()
		spinner.SetHAlign(gtk.AlignCenter)
		spinner.Start()
		vbox.Append(spinner)
		vbox.Append(loadingLabel)

		// Dropdowns for selection
		versionLabel := gtk.NewLabel("Versions:")
		versionLabel.SetHAlign(gtk.AlignStart)
		vbox.Append(versionLabel)

		versionBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
		versionBox.SetHAlign(gtk.AlignFill)
		versionBox.SetVAlign(gtk.AlignCenter)
		versionBox.SetMarginTop(4)
		versionBox.SetMarginBottom(4)
		versionBox.SetMarginStart(0)
		versionBox.SetMarginEnd(0)

		versionSearchEntry := gtk.NewSearchEntry()
		versionSearchEntry.SetPlaceholderText("Search versions... (regex supported)")
		versionSearchEntry.SetHExpand(true)
		versionBox.Append(versionSearchEntry)

		versionDropdown := gtk.NewDropDown(gtk.NewStringList([]string{""}), nil)
		versionDropdown.SetHExpand(true)
		versionDropdown.SetSensitive(false)
		versionBox.Append(versionDropdown)

		vbox.Append(versionBox)

		assetLabel := gtk.NewLabel("Assets:")
		assetLabel.SetHAlign(gtk.AlignStart)
		vbox.Append(assetLabel)

		assetBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
		assetBox.SetHAlign(gtk.AlignFill)
		assetBox.SetVAlign(gtk.AlignCenter)
		assetBox.SetMarginTop(4)
		assetBox.SetMarginBottom(4)
		assetBox.SetMarginStart(0)
		assetBox.SetMarginEnd(0)

		assetSearchEntry := gtk.NewSearchEntry()
		assetSearchEntry.SetPlaceholderText("Search assets... (regex supported)")
		assetSearchEntry.SetHExpand(true)
		assetBox.Append(assetSearchEntry)

		assetDropdown := gtk.NewDropDown(gtk.NewStringList([]string{""}), nil)
		assetDropdown.SetHExpand(true)
		assetDropdown.SetSensitive(false)
		assetBox.Append(assetDropdown)
		vbox.Append(assetBox)

		// Move Download button to bottom, outside assetBox
		buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		buttonBox.SetHAlign(gtk.AlignEnd)
		buttonBox.SetMarginTop(20)
		assetDownloadBtn := gtk.NewButtonWithLabel("Download")
		assetDownloadBtn.SetHExpand(false)
		assetDownloadBtn.SetVExpand(false)
		assetDownloadBtn.SetSizeRequest(-1, -1) // Default size
		assetDownloadBtn.SetMarginTop(0)
		assetDownloadBtn.SetMarginBottom(0)
		assetDownloadBtn.SetSensitive(true)
		buttonBox.Append(assetDownloadBtn)
		vbox.Append(buttonBox)

		// Move refreshCacheBtn to the bottom of the vbox
		refreshCacheBtn := gtk.NewButtonWithLabel("Refresh Releases")
		refreshCacheBtn.SetHAlign(gtk.AlignCenter)
		refreshCacheBtn.SetMarginTop(10)
		refreshCacheBtn.SetVExpand(false)

		assetDownloadBtn.ConnectClicked(func() {
			selectedIdx := assetDropdown.Selected()
			if selectedIdx < 0 || int(selectedIdx) >= len(filteredAssets) {
				fmt.Println("No valid asset selected")
				return
			}
			selectedAsset := filteredAssets[selectedIdx]

			fmt.Printf("Downloading asset: %s (ID: %d) for version: %s\n", selectedAsset.Name, selectedAsset.ID, selectedAsset.Version)
			// Here you would implement the actual download logic using selectedAsset.ID
			// Open a file dialog to choose save location
			fileDialog := gtk.NewFileDialog()
			fileDialog.SetTitle("Save ISO File")
			fileDialog.SetAcceptLabel("Save")
			fileDialog.SetModal(true)
			homeDir, err := getHomeDirectory()
			if err == nil && homeDir != "" {
				fileDialog.SetInitialFile(gio.NewFileForPath(filepath.Join(homeDir, selectedAsset.Name)))
			} else {
				fileDialog.SetInitialFile(gio.NewFileForPath(selectedAsset.Name))
			}

			fileDialog.Save(context.Background(), downloadWin, func(res gio.AsyncResulter) {
				file, err := fileDialog.SaveFinish(res)
				if err != nil {
					fmt.Println("Failed to open file dialog:", err)
					return
				}
				if file == nil {
					fmt.Println("No file selected")
					return
				}
				fmt.Println("Selected file to save:", file.Path())

				// Remove all children from vbox (Gtk4 doesn't have ForEach, use Remove on each child)
				for {
					child := vbox.FirstChild()
					if child == nil {
						break
					}
					vbox.Remove(child)
				}

				// Create a new box for download progress
				progressBox := gtk.NewBox(gtk.OrientationVertical, 10)
				progressBox.SetHAlign(gtk.AlignCenter)
				progressBox.SetVAlign(gtk.AlignCenter)
				progressBox.SetMarginTop(40)
				progressBox.SetMarginBottom(40)
				progressBox.SetMarginStart(40)
				progressBox.SetMarginEnd(40)

				spinnerDownload := gtk.NewSpinner()
				spinnerDownload.SetHAlign(gtk.AlignCenter)
				spinnerDownload.SetVAlign(gtk.AlignCenter)
				spinnerDownload.Start()
				progressBox.Append(spinnerDownload)

				progress := gtk.NewProgressBar()
				progress.SetVExpand(true)
				progress.SetHExpand(true)
				progress.SetMarginBottom(10)
				progressBox.Append(progress)

				downloadLabel := gtk.NewLabel(fmt.Sprintf("Downloading asset %s", selectedAsset.Name))
				downloadLabel.SetHAlign(gtk.AlignCenter)
				progressBox.Append(downloadLabel)

				vbox.Append(progressBox)

				// Run download simulation in a goroutine so the dialog closes immediately
				go func() {
					for i := 0; i <= 100; i += 10 {
						time.Sleep(500 * time.Millisecond)
						fraction := float64(i) / 100.0
						glib.IdleAdd(func() {
							progress.SetFraction(fraction)
							progress.SetText(fmt.Sprintf("Downloading... %d%%", i))
						})
					}
					glib.IdleAdd(func() {
						spinnerDownload.Stop()
						downloadLabel.SetText("Download complete!")
						progress.SetText("Done!")

						// Show the full path of the ISO file
						filePathLabel := gtk.NewLabel(fmt.Sprintf("ISO saved to: %s", file.Path()))
						filePathLabel.SetHAlign(gtk.AlignCenter)
						filePathLabel.SetMarginTop(20)
						progressBox.Append(filePathLabel)

						// Create a 'Go Back' button
						goBackBtn := gtk.NewButtonWithLabel("Go Back")
						goBackBtn.SetHAlign(gtk.AlignCenter)
						goBackBtn.SetMarginTop(20)
						progressBox.Append(goBackBtn)

						goBackBtn.ConnectClicked(func() {
							// Remove all children from vbox
							for {
								child := vbox.FirstChild()
								if child == nil {
									break
								}
								vbox.Remove(child)
							}
							// TODO: Recreate the original UI (or close the window)
							// For now, just close the window:
							downloadWin.Close()
						})
					})
				}()
			})
		})

		// Helper to update dropdowns after fetching assets
		var releaseAssets []ReleaseAsset // Store assets for dropdown logic
		// Update lastVersionList after fetching versions
		updateReleaseDropdowns := func(assets []ReleaseAsset, err error) {
			spinner.Stop()
			if err != nil || len(assets) == 0 {
				fmt.Println("Failed to load releases:", err)
				loadingLabel.SetText("Failed to load releases or no assets found.")
				return
			}
			loadingLabel.SetText("")
			releaseAssets = assets // Save for later use
			versionSet := make(map[string]struct{})
			for _, a := range assets {
				versionSet[a.Version] = struct{}{}
			}
			// Sort versions by semver descending
			var semverVersions []*semver.Version
			versionMap := make(map[string]*semver.Version)
			for v := range versionSet {
				ver, err := semver.NewVersion(v)
				if err == nil {
					semverVersions = append(semverVersions, ver)
					versionMap[v] = ver
				}
			}
			sort.Sort(sort.Reverse(semver.Collection(semverVersions)))
			versions := make([]string, 0, len(semverVersions))
			for _, ver := range semverVersions {
				versions = append(versions, ver.Original())
			}
			lastVersionList = versions // Save for filtering
			versionDropdown.SetModel(gtk.NewStringList(versions))
			versionDropdown.SetSensitive(true)
			if len(versions) > 0 {
				versionDropdown.SetSelected(0)
				latestVersion := versions[0]
				filteredAssets = nil
				var assetList []string
				for _, a := range assets {
					if a.Version == latestVersion && strings.HasSuffix(a.Name, ".iso") {
						assetList = append(assetList, a.Name)
						filteredAssets = append(filteredAssets, a)
					}
				}
				assetDropdown.SetModel(gtk.NewStringList(assetList))
				assetDropdown.SetSensitive(true)
				// Set the number in the label
				versionLabel.SetText(fmt.Sprintf("Versions (%d):", len(versions)))
			}
		}

		// Connect versionDropdown to update assetDropdown
		versionDropdown.Connect("notify::selected", func() {
			selectedObj := versionDropdown.Model().Item(versionDropdown.Selected())
			// This should be a GtkStringObject
			selectedStr, ok := selectedObj.Cast().(*gtk.StringObject)
			if !ok {
				fmt.Println("Selected item is not a GtkStringObject")
				return
			}
			selectedVersion := selectedStr.String()
			// Update assetDropdown based on selected version
			if releaseAssets == nil {
				fmt.Println("No release assets loaded yet")
				return
			}
			filteredAssets = nil
			assetList := []string{}
			for _, a := range releaseAssets {
				if a.Version == selectedVersion && strings.HasSuffix(a.Name, ".iso") {
					assetList = append(assetList, a.Name)
					filteredAssets = append(filteredAssets, a)
				}
			}
			if len(assetList) == 0 {
				fmt.Println("No assets found for version:", selectedVersion)
				assetDropdown.SetModel(gtk.NewStringList([]string{"No assets available"}))
				assetDropdown.SetSensitive(false)
				return
			}
			// Sort asset names
			sort.Strings(assetList)
			assetDropdown.SetModel(gtk.NewStringList(assetList))
			assetDropdown.SetSensitive(true)
		})

		// Helper to keep last asset list for filtering
		var lastAssetList []string

		// Update asset dropdown when version changes
		versionDropdown.Connect("notify::selected", func() {
			selectedObj := versionDropdown.Model().Item(versionDropdown.Selected())
			selectedStr, ok := selectedObj.Cast().(*gtk.StringObject)
			if !ok {
				return
			}
			selectedVersion := selectedStr.String()
			var assetList []string
			for _, a := range releaseAssets {
				if a.Version == selectedVersion && strings.HasSuffix(a.Name, ".iso") {
					assetList = append(assetList, a.Name)
				}
			}
			lastAssetList = assetList
			// Apply search filter if any
			search := strings.ToLower(assetSearchEntry.Text())
			var filtered []string
			for _, name := range assetList {
				if search == "" || strings.Contains(strings.ToLower(name), search) {
					filtered = append(filtered, name)
				}
			}
			if len(filtered) == 0 {
				assetDropdown.SetModel(gtk.NewStringList([]string{"No assets available"}))
				assetDropdown.SetSensitive(false)
			} else {
				assetDropdown.SetModel(gtk.NewStringList(filtered))
				assetDropdown.SetSensitive(true)
			}
			// Update asset label with count
			assetLabel.SetText(fmt.Sprintf("Assets (%d):", len(filtered)))
		})

		// Filter asset dropdown on search entry change
		assetSearchEntry.Connect("search-changed", func() {
			search := strings.ToLower(assetSearchEntry.Text())
			var filtered []string
			var re *regexp.Regexp
			var err error
			if search == "" {
				// empty search, show all assets
				assetDropdown.SetModel(gtk.NewStringList(lastAssetList))
				assetDropdown.SetSensitive(true)
				return
			}
			for _, name := range lastAssetList {
				re, err = regexp.Compile(search)
				if re == nil && err != nil {
					// simple search
					if strings.Contains(strings.ToLower(name), search) {
						filtered = append(filtered, name)
					}
				} else {
					// regex search
					if re.MatchString(strings.ToLower(name)) {
						filtered = append(filtered, name)
					}
				}
			}
			if len(filtered) == 0 {
				assetDropdown.SetModel(gtk.NewStringList([]string{"No assets available"}))
				assetDropdown.SetSensitive(false)
			} else {
				assetDropdown.SetModel(gtk.NewStringList(filtered))
				assetDropdown.SetSensitive(true)
			}
			// Update asset label with count
			assetLabel.SetText(fmt.Sprintf("Assets (%d):", len(filtered)))
		})

		// Filter version dropdown on search entry change
		// connect to search-changed with adds a delay otherwise the trigger is instant
		versionSearchEntry.Connect("search-changed", func() {
			search := strings.ToLower(versionSearchEntry.Text())
			var filtered []string
			var re *regexp.Regexp
			var err error
			if search == "" {
				// empty search, show all versions
				versionDropdown.SetModel(gtk.NewStringList(lastVersionList))
				versionDropdown.SetSensitive(true)
				return
			}
			for _, v := range lastVersionList {
				re, err = regexp.Compile(search)
				if re == nil && err != nil {
					fmt.Println("Using simple search for filtering:", search)
					if strings.Contains(strings.ToLower(v), search) {
						filtered = append(filtered, v)
					}
				} else {
					fmt.Println("Using regex for filtering:", re)
					if re.MatchString(strings.ToLower(v)) {
						filtered = append(filtered, v)
					}
				}
			}
			if len(filtered) == 0 {
				versionDropdown.SetModel(gtk.NewStringList([]string{"No versions found"}))
				versionDropdown.SetSensitive(false)
			} else {
				versionDropdown.SetModel(gtk.NewStringList(filtered))
				versionDropdown.SetSensitive(true)
				versionDropdown.SetSelected(0)
			}
		})

		// Remove previous vbox.Append(refreshCacheBtn)
		// Instead, add a spacer and then append the button
		spacer := gtk.NewBox(gtk.OrientationVertical, 0)
		spacer.SetVExpand(true)
		vbox.Append(spacer)
		vbox.Append(refreshCacheBtn)

		// Helper to set button sensitivity during fetch
		setRefreshBtnActive := func(active bool) {
			refreshCacheBtn.SetSensitive(active)
		}

		refreshCacheBtn.ConnectClicked(func() {
			setRefreshBtnActive(false)
			spinner.Start()
			loadingLabel.SetText("Refreshing releases...")
			versionDropdown.SetSensitive(false)
			assetDropdown.SetSensitive(false)
			go func() {
				ctx := context.Background()
				cacheFile := filepath.Join(os.TempDir(), "kairos_releases_cache.json")
				_ = os.Remove(cacheFile)
				assets, err := GetCachedReleaseAssets(ctx, "kairos-io", "kairos")
				glib.IdleAdd(func() {
					updateReleaseDropdowns(assets, err)
					setRefreshBtnActive(true)
				})
			}()
		})

		// Start fetching release assets in background
		setRefreshBtnActive(false)
		go func() {
			ctx := context.Background()
			assets, err := GetCachedReleaseAssets(ctx, "kairos-io", "kairos")
			glib.IdleAdd(func() {
				updateReleaseDropdowns(assets, err)
				setRefreshBtnActive(true)
			})
		}()
		downloadWin.SetChild(vbox)
		downloadWin.SetVisible(true)
	})

	return downloadBtn
}
