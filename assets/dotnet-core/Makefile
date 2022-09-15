all: cflinuxfs cflinuxfs4

cflinuxfs3:
	dotnet publish -f net6.0 -c Release --sc -r ubuntu.18.04-x64 -o ./cflinuxfs3

cflinuxfs4:
	dotnet publish -f net6.0 -c Release --sc -r ubuntu.22.04-x64 -o ./cflinuxfs4

.PHONY: \
	all
