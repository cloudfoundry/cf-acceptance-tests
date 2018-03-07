:: msbuild must be in path
SET PATH=%PATH%;%WINDIR%\Microsoft.NET\Framework64\v4.0.30319
where msbuild
if errorLevel 1 ( echo "msbuild was not found on PATH" && exit /b 1 )

rmdir /S /Q packages
.nuget\nuget restore || exit /b 1
MSBuild Nora.sln /t:Rebuild /p:Configuration=Release || exit /b 1
packages\nspec.0.9.68\tools\NSpecRunner.exe Nora.Tests\bin\Release\Nora.Tests.dll || exit /b 1
