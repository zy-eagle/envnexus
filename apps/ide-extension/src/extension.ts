import * as vscode from "vscode";
import { disposeDeviceAuth, hasUsableSession, login } from "./auth";
import { sync } from "./sync";
import { checkForUpdates } from "./update";

export function activate(context: vscode.ExtensionContext): void {
  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  context.subscriptions.push(status);

  const refreshStatus = async (): Promise<void> => {
    const signedIn = await hasUsableSession(context);
    if (signedIn) {
      status.text = "$(plug) EnvNexus";
      status.tooltip = "EnvNexus connected. Click to sync now.";
      status.command = "envnexus.sync";
    } else {
      status.text = "$(sign-in) EnvNexus";
      status.tooltip = "EnvNexus not signed in. Click to login.";
      status.command = "envnexus.login";
    }
    status.show();
  };

  context.subscriptions.push(
    vscode.commands.registerCommand("envnexus.login", async () => {
      status.text = "$(sync~spin) EnvNexus";
      status.tooltip = "EnvNexus login in progress...";
      status.command = undefined;
      status.show();
      try {
        await login(context);
      } finally {
        await refreshStatus();
      }
    })
  );
  context.subscriptions.push(
    vscode.commands.registerCommand("envnexus.sync", async () => {
      status.text = "$(sync~spin) EnvNexus";
      status.tooltip = "EnvNexus sync in progress...";
      status.command = undefined;
      status.show();
      try {
        await sync(context);
      } finally {
        await refreshStatus();
      }
    })
  );
  context.subscriptions.push(
    vscode.window.onDidChangeWindowState(() => {
      void refreshStatus();
    })
  );

  void refreshStatus();

  // Check for updates shortly after activation
  setTimeout(() => {
    void checkForUpdates(context);
  }, 5000);
}

export function deactivate(): void {
  disposeDeviceAuth();
}
