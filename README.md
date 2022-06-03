# FabricPing

Network debugging tools for [Service Fabric](https://azure.microsoft.com/en-us/services/service-fabric/)

## Install


### Windows

 * powershell

    ```
    Invoke-WebRequest -OutFile 'FabricPing_windows_amd64.zip' -Uri 'https://github.com/tg123/FabricPing/releases/latest/download/FabricPing_windows_amd64.zip' -UseBasicParsing

    Expand-Archive ./FabricPing_windows_amd64.zip -DestinationPath .
    ```
 

 * using [built in curl](https://docs.microsoft.com/en-us/virtualization/community/team-blog/2017/20171219-tar-and-curl-come-to-windows) in case of `Invoke-WebRequest` not working on Windows Server Core

    ```
    curl.exe -L https://github.com/tg123/FabricPing/releases/latest/download/FabricPing_windows_amd64.zip -o FabricPing_windows_amd64.zip
    ```
 

### Linux

```
curl -L https://github.com/tg123/FabricPing/releases/latest/download/FabricPing_linux_amd64.tar.gz | tar xz
```

## Usage

### Test Fabric protocol endpoints

This mode works with Fabric Port (typically 1025) and Fabric Gateway Port (typically 19000)

```
FabricPing.exe 127.0.0.1:1025
```

### Test Lease endpoint (`-l`)

The mode pings a Lease Port (typically 1026) and requires `FabricPing` running inside the VNET of the Service Fabric Cluster as remote lease agents will connect back

```
FabricPing.exe -l 127.0.0.1:1026
```

### Discover all known nodes (`-d`)

The mode connects to Fabric Port (typically 1025) and requires `FabricPing` running inside the VNET of the Service Fabric Cluster as remote fabric will connect back,
 
```
FabricPing.exe -d 127.0.0.1:1025
```

#### Node Phases
  * Booting: the node is sending VotePing to seed nodes
  * Joining: the node is establishing lease with its neighbors
  * Inserting: the node is negotiating token range with its neighbors
  * Routing: the node is serving
  * Shutdown: the node is shutting down

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft 
trademarks or logos is subject to and must follow 
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
