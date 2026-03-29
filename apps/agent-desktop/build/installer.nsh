; Custom NSIS installer script for EnvNexus Agent
; Copies agent.enx from the installer's directory to the installation directory

!macro customInstall
  ; Look for agent.enx next to the installer executable
  IfFileExists "$EXEDIR\agent.enx" 0 +2
    CopyFiles /SILENT "$EXEDIR\agent.enx" "$INSTDIR\agent.enx"
!macroend
