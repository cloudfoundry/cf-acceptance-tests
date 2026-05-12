all: cflinuxfs4 cflinuxfs5

cflinuxfs4:
	dotnet publish -f net6.0 -c Release --sc -r ubuntu.22.04-x64 -o ./cflinuxfs4

cflinuxfs5:
	dotnet publish -f net8.0 -c Release --sc -r ubuntu.24.04-x64 -o ./cflinuxfs5

.PHONY: \
	all
