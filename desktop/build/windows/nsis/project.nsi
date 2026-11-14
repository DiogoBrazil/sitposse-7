Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
## 
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the wails_tools.nsh file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "my-project" # Default "central"
## !define INFO_COMPANYNAME    "My Company" # Default "7º BPM/PMRO"
## !define INFO_PRODUCTNAME    "My Product Name" # Default "SITPOSSE-7 Central"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "0.1.0"
## !define INFO_COPYRIGHT      "(c) Now, My Company" # Default "© 2026, My Company"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
## !define WAILS_INSTALL_SCOPE     "user"             # Default "machine" - set to "user" for per-user install ($LOCALAPPDATA) without UAC prompt
####
## Include the wails tools
####
# --- SITPOSSE-7: personalizações (wails_tools.nsh é gerado; não editar lá) ---
!define INFO_PROJECTNAME    "sitposse7-central"
!define INFO_PRODUCTVERSION "0.0.1"
!define INFO_COPYRIGHT      "© 2026 7º BPM/PMRO"
!define SITPOSSE_PORTA      "38807"
!define SITPOSSE_REGRA_FW   "SITPOSSE-7 Central (sincronizacao local)"
# WebView2: o bootstrapper padrão do Wails exige internet (RNF-15 exige instalação
# offline). Usamos o "Evergreen Standalone Installer" x64, colocado nesta pasta como
# MicrosoftEdgeWebView2RuntimeInstallerX64.exe. Para builds de desenvolvimento sem ele,
# passe -DSITPOSSE_WEBVIEW2_ONLINE ao makensis.
!ifndef SITPOSSE_WEBVIEW2_ONLINE
    !define WAILS_SKIP_WEBVIEW2
!endif
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "PortugueseBR" # Idioma do instalador

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show # This will always show the installation details.

!macro sitposse.webview2offline
    !ifndef SITPOSSE_WEBVIEW2_ONLINE
    !if /FileExists "MicrosoftEdgeWebView2RuntimeInstallerX64.exe"
        SetRegView 64
        ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
        ${If} $0 == ""
            ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
        ${EndIf}
        ${If} $0 == ""
            DetailPrint "Instalando WebView2 Runtime (offline)..."
            InitPluginsDir
            SetOutPath "$pluginsdir\webview2"
            File "MicrosoftEdgeWebView2RuntimeInstallerX64.exe"
            ExecWait '"$pluginsdir\webview2\MicrosoftEdgeWebView2RuntimeInstallerX64.exe" /silent /install' $1
            ${If} $1 != 0
                MessageBox MB_ICONSTOP "Falha ao instalar o WebView2 Runtime (codigo $1). A Central nao funcionara sem ele."
                Abort
            ${EndIf}
        ${EndIf}
    !else
        !error "Coloque MicrosoftEdgeWebView2RuntimeInstallerX64.exe (Evergreen Standalone x64) em build/windows/nsis/ ou use -DSITPOSSE_WEBVIEW2_ONLINE."
    !endif
    !endif
!macroend

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime
    !insertmacro sitposse.webview2offline

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    
    !insertmacro wails.writeUninstaller

    # Regra de firewall: só rede local, só o executável da Central, perfis Privado e
    # Público (hotspots costumam ser classificados como Públicos). Ver rede-em-campo.md.
    # O texto do netsh NÃO vai para o log: redirecionado, ele sai em UTF-8 e o nsExec o
    # mostra com acentos quebrados (como ANSI ou, com /OEM, como OEM; testado em 13 e
    # 14/11). O log usa mensagens próprias, decididas pelo código de saída.
    nsExec::Exec 'netsh advfirewall firewall delete rule name="${SITPOSSE_REGRA_FW}"'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Regra de firewall anterior removida."
    ${EndIf}
    nsExec::Exec 'netsh advfirewall firewall add rule name="${SITPOSSE_REGRA_FW}" dir=in action=allow program="$INSTDIR\${PRODUCT_EXECUTABLE}" enable=yes profile=private,public protocol=TCP localport=${SITPOSSE_PORTA} remoteip=localsubnet'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Regra de firewall criada: entrada TCP ${SITPOSSE_PORTA}, só da rede local."
    ${Else}
        DetailPrint "Falha ao criar a regra de firewall (código $0)."
        MessageBox MB_ICONEXCLAMATION "Não foi possível criar a regra de firewall da Central (código $0).$\r$\n$\r$\nOs celulares podem não conseguir se conectar. Ao abrir a Central, permita o acesso quando o Windows perguntar, marcando redes públicas." /SD IDOK
    ${EndIf}
SectionEnd

Section "uninstall" 
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath
    # Os DADOS da Central (%LOCALAPPDATA%\SITPOSSE-7) NUNCA são removidos pelo desinstalador.
    nsExec::Exec 'netsh advfirewall firewall delete rule name="${SITPOSSE_REGRA_FW}"'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Regra de firewall removida."
    ${Else}
        DetailPrint "Nenhuma regra de firewall da Central para remover (código $0)."
    ${EndIf}

    RMDir /r $INSTDIR
    # Pasta da empresa (ex.: "7º BPMPMRO"): sem /r, só é removida se ficou vazia.
    RMDir "$INSTDIR\.."

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
