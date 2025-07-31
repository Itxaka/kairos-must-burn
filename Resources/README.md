This folder contains resources used by the application, such as images, icons, and other assets.

# Resources
 - kairos-must-burn.app: MacOs skeleton app
 - icons.icns: Application icon in ICNS format, for macOS
 - kairos-must-burn.ico: Source icon in ICO format, for Windows
 - kairos-must-burn.png: Logo in PNG format

In the root folder there is the rsrc_windows_amd64.syso which is a Windows resource file that contains the application icon and other resources for the Windows build. Golang will automatically include this file when building the Windows version of the application.