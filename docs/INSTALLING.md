# Installing Spotfile (unsigned release)

Spotfile is currently distributed as an unsigned binary. Both macOS and Windows
require a one-time step to allow it to run.

---

## macOS

macOS Gatekeeper quarantines apps downloaded from the internet. Run **one** of
the following after dragging `spotfile.app` to `/Applications` (or wherever you
prefer):

### Option A — right-click open (no Terminal needed)

1. Right-click (or Control-click) `spotfile.app` in Finder.
2. Choose **Open**.
3. Click **Open** in the dialog that appears.
4. After the first launch Gatekeeper remembers the choice permanently.

### Option B — remove the quarantine attribute

```bash
xattr -dr com.apple.quarantine /Applications/spotfile.app
```

Then double-click normally.

### Option C — allow in System Settings

*System Settings → Privacy & Security → scroll to the "spotfile was blocked"
banner → click **Open Anyway**.*

---

## Windows

Windows SmartScreen blocks executables from unknown publishers. When you see
the blue SmartScreen dialog:

1. Click **More info**.
2. Click **Run anyway**.

To disable SmartScreen prompts permanently for a single file (PowerShell, run
as Administrator):

```powershell
Unblock-File -Path "C:\path\to\spotfile.exe"
```

---

## Verifying the download

SHA-256 checksums for each release are listed on the
[Releases page](https://github.com/spotfile-app/spotfile/releases).

```bash
# macOS
shasum -a 256 spotfile-macos.zip

# Windows (PowerShell)
Get-FileHash spotfile-windows-amd64.exe -Algorithm SHA256
```
