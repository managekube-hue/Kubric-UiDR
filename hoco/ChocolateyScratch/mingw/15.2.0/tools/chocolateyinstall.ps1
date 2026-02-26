
$install = Join-Path "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)" "install"
$packageArgs = @{
	packageName   = $env:ChocolateyPackageName
	unzipLocation = $install
	fileType      = 'zip'
	url           = 'https://github.com/niXman/mingw-builds-binaries/releases/download/15.2.0-rt_v13-rev0/i686-15.2.0-release-posix-dwarf-ucrt-rt_v13-rev0.7z'
	url64bit      = 'https://github.com/niXman/mingw-builds-binaries/releases/download/15.2.0-rt_v13-rev0/x86_64-15.2.0-release-posix-seh-ucrt-rt_v13-rev0.7z'
	checksum      = '8559c1e27c48f139400b4d7da848ca30a7bdb9f1fbe06e241b1b53baa6149d2a'
	checksumType  = 'sha256'
	checksum64    = '05b3361a3a3e20a5789db37fc2bb6a70d0b02e4667b5a31d9a6672ea40d22c69'
	checksumType64= 'sha256'

}

New-Item -ItemType Directory -Force -Path $install | Out-Null
Install-ChocolateyZipPackage @packageArgs

Uninstall-BinFile -Name addr2line
Uninstall-BinFile -Name ar
Uninstall-BinFile -Name as
Uninstall-BinFile -Name c++
Uninstall-BinFile -Name c++filt
Uninstall-BinFile -Name cpp
Uninstall-BinFile -Name dlltool
Uninstall-BinFile -Name dllwrap
Uninstall-BinFile -Name dwp
Uninstall-BinFile -Name elfedit
Uninstall-BinFile -Name g++
Uninstall-BinFile -Name gcc-ar
Uninstall-BinFile -Name gcc-nm
Uninstall-BinFile -Name gcc-ranlib
Uninstall-BinFile -Name gcc
Uninstall-BinFile -Name gcov-dump
Uninstall-BinFile -Name gcov-tool
Uninstall-BinFile -Name gcov
Uninstall-BinFile -Name gdb
Uninstall-BinFile -Name gdborig
Uninstall-BinFile -Name gdbserver
Uninstall-BinFile -Name gendef
Uninstall-BinFile -Name genidl
Uninstall-BinFile -Name genpeimg
Uninstall-BinFile -Name gfortran
Uninstall-BinFile -Name gprof
Uninstall-BinFile -Name ld.bfd
Uninstall-BinFile -Name ld
Uninstall-BinFile -Name ld.gold
Uninstall-BinFile -Name lto-dump
Uninstall-BinFile -Name mingw32-make
Uninstall-BinFile -Name nm
Uninstall-BinFile -Name objcopy
Uninstall-BinFile -Name objdump
Uninstall-BinFile -Name ranlib
Uninstall-BinFile -Name readelf
Uninstall-BinFile -Name size
Uninstall-BinFile -Name strings
Uninstall-BinFile -Name strip
Uninstall-BinFile -Name widl
Uninstall-BinFile -Name windmc
Uninstall-BinFile -Name windres
Uninstall-BinFile -Name x86_64-w64-mingw32-c++
Uninstall-BinFile -Name x86_64-w64-mingw32-g++
Uninstall-BinFile -Name x86_64-w64-mingw32-gcc-12.2.0
Uninstall-BinFile -Name x86_64-w64-mingw32-gcc-ar
Uninstall-BinFile -Name x86_64-w64-mingw32-gcc-nm
Uninstall-BinFile -Name x86_64-w64-mingw32-gcc-ranlib
Uninstall-BinFile -Name x86_64-w64-mingw32-gcc
Uninstall-BinFile -Name x86_64-w64-mingw32-gfortran
Uninstall-BinFile -Name cc1
Uninstall-BinFile -Name cc1plus
Uninstall-BinFile -Name collect2
Uninstall-BinFile -Name f951
Uninstall-BinFile -Name g++-mapper-server
Uninstall-BinFile -Name lto-wrapper
Uninstall-BinFile -Name lto1
Uninstall-BinFile -Name fixincl
Uninstall-BinFile -Name gdbmtool
Uninstall-BinFile -Name gdbm_dump
Uninstall-BinFile -Name gdbm_load
Uninstall-BinFile -Name python3.9
Uninstall-BinFile -Name python3
Uninstall-BinFile -Name python3w
Uninstall-BinFile -Name x86_64-w64-mingw32-captoinfo
Uninstall-BinFile -Name x86_64-w64-mingw32-clear
Uninstall-BinFile -Name x86_64-w64-mingw32-infocmp
Uninstall-BinFile -Name x86_64-w64-mingw32-infotocap
Uninstall-BinFile -Name x86_64-w64-mingw32-reset
Uninstall-BinFile -Name x86_64-w64-mingw32-tabs
Uninstall-BinFile -Name x86_64-w64-mingw32-tic
Uninstall-BinFile -Name x86_64-w64-mingw32-toe
Uninstall-BinFile -Name x86_64-w64-mingw32-tput
Uninstall-BinFile -Name x86_64-w64-mingw32-tset
Uninstall-BinFile -Name python
Uninstall-BinFile -Name pythonw
Uninstall-BinFile -Name ar
Uninstall-BinFile -Name as
Uninstall-BinFile -Name dlltool
Uninstall-BinFile -Name ld.bfd
Uninstall-BinFile -Name ld
Uninstall-BinFile -Name ld.gold
Uninstall-BinFile -Name nm
Uninstall-BinFile -Name objcopy
Uninstall-BinFile -Name objdump
Uninstall-BinFile -Name ranlib
Uninstall-BinFile -Name readelf
Uninstall-BinFile -Name strip

If (-Not (Test-Path $env:ProgramData)) {
  $ProgramData = "C:\ProgramData"
} else {
  $ProgramData = $env:ProgramData
}

Write-Output "Using ProgramData: $ProgramData"

Move-Item $install "$ProgramData\mingw64"

$("mingw32", "mingw64") | ForEach {
  $bin = (Join-Path "$ProgramData\mingw64" (Join-Path $_ "bin"))
  If (Test-Path $bin) {
    Write-Output "Testing path: $bin...Found!"
    Install-ChocolateyPath $bin -PathType 'Machine'
  } else {
    Write-Output "Testing path: $bin...Not Found!"
  }
}

