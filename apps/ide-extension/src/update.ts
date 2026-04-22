import * as os from "os";
import * as path from "path";
import * as fs from "fs/promises";
import * as vscode from "vscode";
import fetch from "node-fetch";
import * as semver from "semver";
import { getApiBase } from "./auth";

interface ExtensionUpdateResponse {
  version: string;
  download_url: string;
}

export async function checkForUpdates(context: vscode.ExtensionContext): Promise<void> {
  try {
    const currentVersion = context.extension.packageJSON.version;
    const res = await fetch(`${getApiBase()}/ide-sync/extension/latest`, {
      method: "GET",
      headers: { Accept: "application/json" }
    });
    if (!res.ok) {
      return;
    }
    const body = (await res.json()) as { data: ExtensionUpdateResponse };
    if (!body.data || !body.data.version || !body.data.download_url) {
      return;
    }

    const latestVersion = body.data.version;
    if (semver.gt(latestVersion, currentVersion)) {
      const choice = await vscode.window.showInformationMessage(
        `EnvNexus Sync: A new version (${latestVersion}) is available. (Current: ${currentVersion})`,
        "Update Now",
        "Later"
      );
      if (choice === "Update Now") {
        await downloadAndInstall(body.data.download_url);
      }
    }
  } catch (e) {
    console.error("EnvNexus Sync auto-update check failed:", e);
  }
}

async function downloadAndInstall(url: string): Promise<void> {
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: "Downloading EnvNexus Sync update...",
      cancellable: false
    },
    async () => {
      try {
        const res = await fetch(url);
        if (!res.ok) {
          throw new Error(`Failed to download: ${res.statusText}`);
        }
        const buffer = await res.buffer();
        const tmpPath = path.join(os.tmpdir(), `envnexus-sync-update-${Date.now()}.vsix`);
        await fs.writeFile(tmpPath, buffer);

        // Install the extension
        await vscode.commands.executeCommand("workbench.extensions.installExtension", vscode.Uri.file(tmpPath));

        const reloadChoice = await vscode.window.showInformationMessage(
          "EnvNexus Sync updated successfully! Please reload the window to apply changes.",
          "Reload Window"
        );
        if (reloadChoice === "Reload Window") {
          await vscode.commands.executeCommand("workbench.action.reloadWindow");
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        void vscode.window.showErrorMessage(`Failed to update EnvNexus Sync: ${msg}`);
      }
    }
  );
}
