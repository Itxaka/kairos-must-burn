package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed Resources/kairos-must-burn.png
var logoData []byte
var lastVersionList []string

func main() {
	f, err := os.CreateTemp("", "logo.png")
	if err != nil {
		panic("Failed to create temporary logo file: " + err.Error())
	}
	defer os.Remove(f.Name()) // Clean up temporary file
	if _, err := f.Write(logoData); err != nil {
		panic("Failed to write logo data to temporary file: " + err.Error())
	}
	f.Close() // Close the file to ensure it's written

	app := gtk.NewApplication("com.kairos.isoburn", gio.ApplicationHandlesOpen|gio.ApplicationDefaultFlags)

	app.ConnectActivate(func() {
		win := gtk.NewApplicationWindow(app)
		win.SetTitle("Kairos Must Burn")

		// Check for elevated permissions immediately
		hasPermissions, err := CheckElevatedPermissions()
		if !hasPermissions {
			// Create permission error dialog
			dialog := gtk.NewDialog()
			dialog.SetTitle("Elevated Permissions Required")
			dialog.SetTransientFor(&win.Window)
			dialog.SetModal(true)

			contentArea := dialog.ContentArea()
			box := gtk.NewBox(gtk.OrientationVertical, 10)
			box.SetMarginTop(20)
			box.SetMarginBottom(20)
			box.SetMarginStart(20)
			box.SetMarginEnd(20)

			icon := gtk.NewImageFromIconName("dialog-warning")
			icon.SetPixelSize(48)
			icon.SetMarginBottom(10)
			box.Append(icon)

			msgLabel := gtk.NewLabel("This application requires elevated permissions to write to USB devices.")
			msgLabel.SetHAlign(gtk.AlignCenter)
			msgLabel.SetWrap(true)
			msgLabel.SetMarginBottom(10)
			box.Append(msgLabel)

			var errorMsg string
			if err != nil {
				errorMsg = "Error: " + err.Error()
			} else {
				if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
					errorMsg = "Please run this application with sudo."
				} else if runtime.GOOS == "windows" {
					errorMsg = "Please right-click and select 'Run as administrator'."
				} else {
					errorMsg = "Please run this application with elevated privileges."
				}
			}

			errorLabel := gtk.NewLabel(errorMsg)
			errorLabel.SetHAlign(gtk.AlignCenter)
			errorLabel.SetWrap(true)
			box.Append(errorLabel)

			contentArea.Append(box)

			// Create a button box with proper margins and styling
			buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
			buttonBox.SetMarginTop(20)
			buttonBox.SetMarginBottom(20)
			buttonBox.SetMarginStart(20)
			buttonBox.SetMarginEnd(20)
			buttonBox.SetHAlign(gtk.AlignCenter)

			// Create exit button with styling
			exitBtn := gtk.NewButton()
			exitBtn.SetLabel("Exit")
			exitBtn.SetCSSClasses([]string{"suggested-action"}) // Add styling to make button more prominent
			exitBtn.SetMarginTop(10)
			exitBtn.SetMarginBottom(10)
			exitBtn.SetMarginStart(40)
			exitBtn.SetMarginEnd(40)

			buttonBox.Append(exitBtn)
			contentArea.Append(buttonBox)

			// Show dialog
			dialog.SetVisible(true)
			dialog.Present()
			dialog.Focus()

			// Connect exit button click
			exitBtn.ConnectClicked(func() {
				app.Quit()
			})

			// Also handle dialog close
			dialog.Connect("close-request", func() bool {
				app.Quit()
				return false
			})

			return
		}

		burnBtn := gtk.NewButtonWithLabel("🔥 Burn!")
		burnBtn.SetSensitive(false)

		var isoPath string
		var drive string

		isoBtn := gtk.NewButtonWithLabel("💿 Select ISO")
		isoBtn.ConnectClicked(func() {
			dialog := gtk.NewFileDialog()
			dialog.SetTitle("Select ISO File")
			dialog.SetModal(true)

			// More reliable way to get home directory when running with elevated permissions
			homeDir, err := getHomeDirectory()
			if err == nil && homeDir != "" {
				// Create a file for the home directory
				gfile := gio.NewFileForPath(homeDir)
				if gfile != nil {
					dialog.SetInitialFolder(gfile)
				}
			}

			// Create and apply filter for ISO files
			filter := gtk.NewFileFilter()
			filter.SetName("ISO files")
			filter.AddPattern("*.iso")
			filter.AddMIMEType("application/x-iso9660-image")
			dialog.SetDefaultFilter(filter)

			dialog.Open(context.Background(), &win.Window, func(res gio.AsyncResulter) {
				file, err := dialog.OpenFinish(res)
				if err == nil && file != nil {
					isoPath = file.Path()
					isoBtn.SetLabel("ISO: " + isoPath)
					if drive != "" {
						burnBtn.SetSensitive(true)
					} else {
						burnBtn.SetSensitive(false)
					}

				}
			})
		})

		drives := ListUSBDrives()
		model := gtk.NewStringList(drives)
		driveDropdown := gtk.NewDropDown(model, nil)
		driveDropdown.SetSelected(0)

		refreshBtn := gtk.NewButtonWithLabel("⟳")
		refreshBtn.SetTooltipText("Refresh USB drives list")
		refreshBtn.SetHAlign(gtk.AlignStart)
		refreshBtn.SetVAlign(gtk.AlignCenter)
		refreshBtn.SetSizeRequest(40, 32)
		refreshBtn.ConnectClicked(func() {
			newDrives := ListUSBDrives()
			model := gtk.NewStringList(newDrives)
			driveDropdown.SetModel(model)
			driveDropdown.SetSelected(0)
			drives = newDrives
			burnBtn.SetSensitive(false)
		})

		driveBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
		driveDropdown.SetHExpand(true) // Make dropdown expand to fill available space
		driveBox.Append(driveDropdown)
		driveBox.Append(refreshBtn)

		driveDropdown.Connect("notify::selected", func() {
			index := int(driveDropdown.Selected())
			if index > 0 && index < len(drives) {
				drive = drives[index]
				if isoPath != "" {
					burnBtn.SetSensitive(true)
				} else {
					burnBtn.SetSensitive(false)
				}
			} else {
				burnBtn.SetSensitive(false)
			}
		})

		// Add image at the top and make it bigger
		logo := gtk.NewImageFromFile(f.Name())
		logo.SetPixelSize(256) // Make the image bigger
		layout := gtk.NewBox(gtk.OrientationVertical, 10)
		layout.SetMarginTop(20)
		layout.SetMarginBottom(20)
		layout.SetMarginStart(20)
		layout.SetMarginEnd(20)
		layout.Append(logo)
		layout.Append(isoBtn)

		// Add "Download ISOs" button below isoBtn
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
			versionLabel := gtk.NewLabel("Version:")
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
			versionSearchEntry.SetPlaceholderText("Search versions...")
			versionSearchEntry.SetHExpand(true)
			versionBox.Append(versionSearchEntry)

			versionDropdown := gtk.NewDropDown(gtk.NewStringList([]string{""}), nil)
			versionDropdown.SetHExpand(true)
			versionDropdown.SetSensitive(false)
			versionBox.Append(versionDropdown)

			vbox.Append(versionBox)

			assetLabel := gtk.NewLabel("Asset:")
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
			assetSearchEntry.SetPlaceholderText("Search assets...")
			assetSearchEntry.SetHExpand(true)
			assetBox.Append(assetSearchEntry)

			assetDropdown := gtk.NewDropDown(gtk.NewStringList([]string{""}), nil)
			assetDropdown.SetHExpand(true)
			assetDropdown.SetSensitive(false)
			assetBox.Append(assetDropdown)

			vbox.Append(assetBox)

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
					assetList := []string{}
					for _, a := range assets {
						if a.Version == latestVersion && strings.HasSuffix(a.Name, ".iso") {
							assetList = append(assetList, a.Name)
						}
					}
					assetDropdown.SetModel(gtk.NewStringList(assetList))
					assetDropdown.SetSensitive(true)
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
				assetList := []string{}
				for _, a := range releaseAssets {
					if a.Version == selectedVersion && strings.HasSuffix(a.Name, ".iso") {
						assetList = append(assetList, a.Name)
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
				assetList := []string{}
				for _, a := range releaseAssets {
					if a.Version == selectedVersion && strings.HasSuffix(a.Name, ".iso") {
						assetList = append(assetList, a.Name)
					}
				}
				lastAssetList = assetList
				// Apply search filter if any
				search := strings.ToLower(assetSearchEntry.Text())
				filtered := []string{}
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
			})

			// Filter asset dropdown on search entry change
			assetSearchEntry.ConnectChanged(func() {
				search := strings.ToLower(assetSearchEntry.Text())
				filtered := []string{}
				for _, name := range lastAssetList {
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
			})

			// Filter version dropdown on search entry change
			versionSearchEntry.ConnectChanged(func() {
				search := strings.ToLower(versionSearchEntry.Text())
				filtered := []string{}
				for _, v := range lastVersionList {
					if search == "" || strings.Contains(strings.ToLower(v), search) {
						filtered = append(filtered, v)
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
			// Move refreshCacheBtn to the bottom of the vbox
			refreshCacheBtn := gtk.NewButtonWithLabel("Refresh Releases")
			refreshCacheBtn.SetHAlign(gtk.AlignCenter)
			refreshCacheBtn.SetMarginTop(10)
			refreshCacheBtn.SetVExpand(false)

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
		layout.Append(downloadBtn)

		layout.Append(driveBox)
		layout.Append(burnBtn)

		win.SetChild(layout)
		win.SetVisible(true)
		win.SetDefaultSize(800, 600)
		win.Present() // Bring window to the foreground and give it focus

		// Function to start the burning process
		startBurning := func() {
			content := gtk.NewBox(gtk.OrientationVertical, 20)
			content.SetMarginTop(30)
			content.SetMarginBottom(30)
			content.SetMarginStart(30)
			content.SetMarginEnd(30)
			content.SetHAlign(gtk.AlignCenter)
			content.SetVAlign(gtk.AlignCenter)

			// Add image above progress bar and make it bigger
			logo = gtk.NewImageFromFile(f.Name())
			logo.SetPixelSize(256) // Make the image bigger
			content.Append(logo)

			progress := gtk.NewProgressBar()
			progress.SetVExpand(true)
			progress.SetHExpand(true)
			progress.SetMarginBottom(10)

			status := gtk.NewLabel("Burning...")
			status.SetMarginBottom(10)
			status.SetHAlign(gtk.AlignCenter)

			exitBtn := gtk.NewButtonWithLabel("Exit")
			exitBtn.SetSensitive(false)
			exitBtn.SetHAlign(gtk.AlignCenter)

			content.Append(progress)
			content.Append(status)
			content.Append(exitBtn)

			win.SetChild(content)

			go func() {
				Burn(isoPath, drive, progress, status, exitBtn)
			}()

			exitBtn.ConnectClicked(func() {
				win.Close()
			})
		}

		burnBtn.ConnectClicked(func() {
			// Check if any partitions of the selected device are mounted
			if drive != "" && !strings.HasPrefix(drive, "Select") && !strings.HasPrefix(drive, "No USB") {
				devPath := strings.Fields(drive)[0] // e.g. /dev/sdb
				mounted, err := IsDeviceMounted(devPath)
				if err == nil && len(mounted) > 0 {
					// Create a custom dialog using available widgets
					dialog := gtk.NewDialog()
					dialog.SetTitle("Unmount Partitions")
					dialog.SetTransientFor(&win.Window)
					dialog.SetModal(true)
					//dialog.SetDefaultWidth(400)

					contentArea := dialog.ContentArea()
					box := gtk.NewBox(gtk.OrientationVertical, 10)
					box.SetMarginTop(20)
					box.SetMarginBottom(20)
					box.SetMarginStart(20)
					box.SetMarginEnd(20)

					// Add message
					msg := "Some partitions are mounted:\n" + strings.Join(mounted, "\n")
					msgLabel := gtk.NewLabel(msg)
					msgLabel.SetHAlign(gtk.AlignStart)
					msgLabel.SetWrap(true)
					box.Append(msgLabel)

					// Add question
					questionLabel := gtk.NewLabel("Do you want to unmount them?")
					questionLabel.SetHAlign(gtk.AlignStart)
					questionLabel.SetMarginTop(10)
					box.Append(questionLabel)

					contentArea.Append(box)

					// Create a button box with proper margins and styling
					buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
					buttonBox.SetMarginTop(20)
					buttonBox.SetMarginBottom(20)
					buttonBox.SetMarginStart(20)
					buttonBox.SetMarginEnd(20)
					buttonBox.SetHAlign(gtk.AlignCenter)

					// Create Cancel button
					cancelBtn := gtk.NewButton()
					cancelBtn.SetLabel("Cancel")
					cancelBtn.SetMarginTop(10)
					cancelBtn.SetMarginBottom(10)
					cancelBtn.SetMarginStart(20)
					cancelBtn.SetMarginEnd(20)

					// Create Unmount button with styling
					unmountBtn := gtk.NewButton()
					unmountBtn.SetLabel("Unmount")
					unmountBtn.SetCSSClasses([]string{"suggested-action"})
					unmountBtn.SetMarginTop(10)
					unmountBtn.SetMarginBottom(10)
					unmountBtn.SetMarginStart(20)
					unmountBtn.SetMarginEnd(20)

					buttonBox.Append(cancelBtn)
					buttonBox.Append(unmountBtn)
					contentArea.Append(buttonBox)

					// Show dialog
					dialog.Show()

					// Connect cancel button click
					cancelBtn.ConnectClicked(func() {
						dialog.Destroy()
					})

					// Connect unmount button click
					unmountBtn.ConnectClicked(func() {
						dialog.Destroy()

						// Try to unmount
						err := UnmountDevice(mounted)
						if err != nil {
							errDialog(win.Window, err)
						} else {
							// Continue with burn after successful unmount
							startBurning()
						}
					})
					return
				}
			}

			// Start burning directly if no unmounting is needed
			startBurning()
		})

	})

	app.Run(os.Args)
}

func errDialog(win gtk.Window, err error) {
	// Show error dialog
	errD := gtk.NewDialog()
	errD.SetTitle("Error")
	errD.SetTransientFor(&win)
	errD.SetModal(true)

	contentArea := errD.ContentArea()
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	// Add warning icon
	icon := gtk.NewImageFromIconName("dialog-error")
	icon.SetPixelSize(48)
	icon.SetMarginBottom(10)
	icon.SetHAlign(gtk.AlignCenter)
	box.Append(icon)

	// Error message
	errMsgLabel := gtk.NewLabel("Failed to unmount: " + err.Error())
	errMsgLabel.SetHAlign(gtk.AlignCenter)
	errMsgLabel.SetWrap(true)
	box.Append(errMsgLabel)

	contentArea.Append(box)

	// Create a button box with proper margins and styling
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	buttonBox.SetMarginTop(20)
	buttonBox.SetMarginBottom(20)
	buttonBox.SetMarginStart(20)
	buttonBox.SetMarginEnd(20)
	buttonBox.SetHAlign(gtk.AlignCenter)

	// Create close button with styling
	closeBtn := gtk.NewButton()
	closeBtn.SetLabel("Close")
	closeBtn.SetCSSClasses([]string{"suggested-action"})
	closeBtn.SetMarginTop(10)
	closeBtn.SetMarginBottom(10)
	closeBtn.SetMarginStart(40)
	closeBtn.SetMarginEnd(40)

	buttonBox.Append(closeBtn)
	contentArea.Append(buttonBox)

	// Show dialog
	errD.Show()

	// Connect close button click
	closeBtn.ConnectClicked(func() {
		errD.Destroy()
	})
}
