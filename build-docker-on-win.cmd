set BUILD_INFO=-ldflags "-X t.GitHash=none -X t.Version=0 -X t.BuildType=dev"
set GOOS=linux
set GOARCH=amd64
set EXTENSION=
set GO111MODULE=on
set CGO_ENABLED=0

go build -o ./bin/otelcol_%GOOS%_%GOARCH%%EXTENSION% %BUILD_INFO% ./cmd/otelcol
if %errorlevel% neq 0 exit /b %errorlevel%

copy .\bin\otelcol_linux_amd64 .\cmd\otelcol\otelcol
if %errorlevel% neq 0 exit /b %errorlevel%

docker build -t otelcol ./cmd/otelcol/
if %errorlevel% neq 0 exit /b %errorlevel%

del .\cmd\otelcol\otelcol
if %errorlevel% neq 0 exit /b %errorlevel%
