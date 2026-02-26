# gen_wpcap_lib.ps1
# Generates wpcap.lib and Packet.lib import libraries from the installed
# npcap DLLs using the VS 2022 Build Tools lib.exe.
#
# Output: vendor/pcap-sdk/x64/wpcap.lib
#         vendor/pcap-sdk/x64/Packet.lib

$ErrorActionPreference = 'Stop'

$npcapDir = "C:\Windows\System32\Npcap"
$outDir   = Join-Path $PSScriptRoot "..\vendor\pcap-sdk\x64"
$libExe   = "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC\14.44.35207\bin\HostX64\x64\lib.exe"

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

# ── Inline C# PE export parser ────────────────────────────────────────────────
Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.IO;
using System.Text;

public static class PEExports {
    static byte[]  _b;
    static short   _numSec;
    static int     _secStart;

    static int RvaToOffset(int rva) {
        for (int i = 0; i < _numSec; i++) {
            int s   = _secStart + i * 40;
            int va  = BitConverter.ToInt32(_b, s + 12);
            int vsz = BitConverter.ToInt32(_b, s + 8);
            int raw = BitConverter.ToInt32(_b, s + 20);
            if (rva >= va && rva < va + vsz) return raw + (rva - va);
        }
        throw new Exception("RVA 0x" + rva.ToString("X") + " not found in sections");
    }

    public static string[] GetExports(string path) {
        _b = File.ReadAllBytes(path);

        if (BitConverter.ToInt16(_b, 0) != 0x5A4D)
            throw new Exception("Not a valid PE file");

        int peOff = BitConverter.ToInt32(_b, 60);
        if (BitConverter.ToInt32(_b, peOff) != 0x00004550)
            throw new Exception("Invalid PE signature");

        short machine = BitConverter.ToInt16(_b, peOff + 4);
        bool  is64    = (machine == unchecked((short)0x8664));
        _numSec       = BitConverter.ToInt16(_b, peOff + 6);
        short optSz   = BitConverter.ToInt16(_b, peOff + 20);
        int   optOff  = peOff + 24;
        int   ddOff   = is64 ? optOff + 112 : optOff + 96;
        int   expRVA  = BitConverter.ToInt32(_b, ddOff);
        if (expRVA == 0) return new string[0];
        _secStart = peOff + 24 + optSz;

        int eOff   = RvaToOffset(expRVA);
        int nNames = BitConverter.ToInt32(_b, eOff + 24);
        int nRVA   = BitConverter.ToInt32(_b, eOff + 32);
        int nOff   = RvaToOffset(nRVA);

        var list = new List<string>();
        for (int i = 0; i < nNames; i++) {
            int nrva = BitConverter.ToInt32(_b, nOff + i * 4);
            int npos = RvaToOffset(nrva);
            var sb   = new StringBuilder();
            while (_b[npos] != 0) sb.Append((char)_b[npos++]);
            list.Add(sb.ToString());
        }
        return list.ToArray();
    }
}
'@

function Build-ImportLib {
    param(
        [string]$DllPath,
        [string]$LibraryName
    )
    $defFile = Join-Path $outDir "$LibraryName.def"
    $libFile = Join-Path $outDir "$LibraryName.lib"

    Write-Host "Extracting exports from $DllPath ..."
    $exports = [PEExports]::GetExports($DllPath)
    Write-Host "  Found $($exports.Length) exported symbols"

    $defLines = @("LIBRARY $LibraryName", "EXPORTS")
    foreach ($sym in ($exports | Sort-Object)) { $defLines += "    $sym" }
    $defLines | Out-File -FilePath $defFile -Encoding ASCII
    Write-Host "  Written: $defFile"

    $result = & $libExe /DEF:"$defFile" /OUT:"$libFile" /MACHINE:X64 2>&1
    if ($LASTEXITCODE -ne 0) { Write-Error "lib.exe failed for ${LibraryName}:`n$result"; exit 1 }
    Write-Host "  Generated: $libFile"
}

Build-ImportLib -DllPath (Join-Path $npcapDir "wpcap.dll")  -LibraryName "wpcap"
Build-ImportLib -DllPath (Join-Path $npcapDir "Packet.dll") -LibraryName "Packet"

Write-Host ""
Write-Host "Done. LIBPCAP_LIBDIR is set in .cargo/config.toml to vendor/pcap-sdk/x64"
