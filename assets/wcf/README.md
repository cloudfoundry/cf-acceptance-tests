## CloudFoundry WCF Service Hosting Example

This sample app is a clone of https://github.com/sneal/WCFServiceSample. We've modified this to work with WATS.

This sample assumes you have an existing Windows cell with .NET 4.5.2 registered as the `windows`
stack in CloudFoundry. You will also need a Windows development machine to compile the solution.

2. Build the solution using VisualStudio or MSBuild. I used [VisualStudio 2015 Community Edition](https://www.visualstudio.com/en-us/products/visual-studio-community-vs.aspx).
3. From the root of the cloned repo where the .sln file is, run: `cf push wcfsample -s windows -b hwc_buildpack --health-check-type none -p ./Hello.Service.IIS/`

Take particular note of the handler mapping in the web.config and the associated .svc file in the web project. This
allows the Hostable Web Core to serve the WCF service in the library project.

```
    <handlers>
      <add name="svc-Integrated" path="*.svc" verb="*" type="System.ServiceModel.Activation.HttpHandler, System.ServiceModel.Activation, Version=4.0.0.0, Culture=neutral, PublicKeyToken=31bf3856ad364e35" />
    </handlers>
```
