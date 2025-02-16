@echo off

if not exist bin (
	mkdir bin
)

echo Building...
set GOOS=linux
go build -o bin\replay

echo Build complete.