# PNG to ICO

A PNG-to-ICO converter built with [MyGo native](https://github.com/egoist/mygo), with GUI and command-line versions. The entire interface is written in Go.

## Usage

Launch the GUI and enter a path under **PNG file**, click **Browse…** to select a PNG, or drop one PNG file into the window. Selecting an image does not start conversion automatically.

**ICO output path** defaults to an ICO with the same name in the source folder, for example `D:\Pictures\logo.png` → `D:\Pictures\logo.ico`. You can edit this path or click its **Browse…** button to choose a destination and filename. The default output path follows changes to the PNG path; a custom output path is preserved.

Confirm both paths, then click **Convert** to generate the ICO. Paths entered manually may include the double quotes added by Windows **Copy as path**.

The output path is shown after a successful conversion; failures display an error message. If the destination ICO already exists, the GUI asks whether to overwrite it. The source PNG is never modified. Canceling a file picker does not start conversion.

Each ICO contains six sizes: 16, 32, 48, 64, 128, and 256 pixels, with transparency preserved. Non-square images are scaled proportionally and padded with a transparent background, without cropping. Smaller sizes use 32-bit DIBs; the 256-pixel size uses PNG compression. Custom sizes and batch conversion are not currently available.

## Command-line usage

Use the separate `bin/png2ico-cli.exe` for terminals and scripts:

```powershell
# Save an ICO with the same name beside the PNG
.\bin\png2ico-cli.exe "D:\Pictures\logo.png"

# Specify an output path (or provide it as the second path)
.\bin\png2ico-cli.exe "D:\Pictures\logo.png" -o "D:\Icons\logo.ico"
.\bin\png2ico-cli.exe "D:\Pictures\logo.png" "D:\Icons\logo.ico"

# Explicitly overwrite an existing ICO
.\bin\png2ico-cli.exe "D:\Pictures\logo.png" -f

# Show help
.\bin\png2ico-cli.exe --help
```

`-o` is equivalent to `--output`, and `-f` to `--force`. Options may appear before or after file paths. The output directory must already exist, and the output file must use the `.ico` extension. The CLI does not open a GUI or an overwrite confirmation dialog; an existing destination file causes conversion to fail by default.

On success, stdout contains only the full path of the generated ICO. Errors are written to stderr. Exit codes are `0` for success or help, `1` for conversion failure, and `2` for invalid arguments. Read the exit code through `$LASTEXITCODE` in PowerShell or `%ERRORLEVEL%` in CMD.

The programs have separate entry points: `png2ico.exe` opens the GUI and does not parse command-line arguments; `png2ico-cli.exe` handles command-line arguments and does not open the GUI. Running the CLI without arguments shows an argument hint and returns exit code `2`. Both share the same PNG-to-ICO conversion logic and embed the same application icon. The CLI does not depend on the MyGo interface module.

## Development and builds

Currently Windows-only (Windows 10/11 recommended). Go 1.27.1 or later is required. Node.js, CGO, and WebView2 are not required.

Before the first build, run `go generate .` to create the `appicon_windows_*.syso` resource files that embed the application icon. Building without them still succeeds, but the executables carry no icon. See [Application icon](#application-icon).

Use the build script to generate a GUI executable that does not create a console window:

```powershell
.\build.ps1
```

Equivalent Go build command:

```powershell
go build -trimpath -tags mygo_no_inspector -ldflags "-H=windowsgui -s -w -X github.com/egoist/mygo.production=1" -o bin/png2ico.exe .
```

Double-click `bin/png2ico.exe` to launch the GUI.

A plain `go build` produces a console-subsystem executable by default. The GUI entry point immediately detaches from the console, but Windows may briefly create a console window before Go code runs. Use the script above or build with `-H=windowsgui` to prevent that flash.

Build the console version (running without arguments shows an argument hint and does not open the GUI):

```powershell
.\build.ps1 -CLI

# Or use Go directly
go build -trimpath -ldflags "-s -w" -o bin/png2ico-cli.exe ./cmd/png2ico-cli
```

## Application icon

Both executables embed `assets/png2ico-icon.ico` through the `appicon_windows_*.syso` files in the main package and `cmd/png2ico-cli`. The GUI executable, window title bar, and taskbar all use this icon. No additional image files need to be placed beside the executables.

The `.syso` files are build inputs produced by code generation: **run `go generate .` before building, or the executables will have no icon**. The build itself does not invoke generation automatically.

```powershell
go generate .
.\build.ps1
```

Regeneration is required after replacing the ICO in `assets/` or deleting the `.syso` files. Resource generation is pinned to `go-winres v0.3.3`; the first run downloads the tool's dependencies. With the `.syso` files present, `go run .` and `go build` embed the icon automatically.
