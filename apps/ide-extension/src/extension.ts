import * as vscode from "vscode";
import { disposeDeviceAuth, login } from "./auth";
import { sync } from "./sync";
import { checkForUpdates } from "./update";

export function activate(context: vscode.ExtensionContext): void {
  context.subscriptions.push(
    vscode.commands.registerCommand("envnexus.login", async () => {
      await login(context);
    })
  );
  context.subscriptions.push(
    vscode.commands.registerCommand("envnexus.sync", async () => {
      await sync(context);
    })
  );

  // Check for updates shortly after activation
  setTimeout(() => {
    void checkForUpdates(context);
  }, 5000);
}

export function deactivate(): void {
  disposeDeviceAuth();
}
