# autopyter
Tool to automatically execute Python code from the clipboard.

`autopyter` is a self-contained binary custom Jupiter Notebook client. You need to have a working Jupyter Notebook server.

# How to use

### Step 1 - Spin up a Jupyter Notebook docker container
You can use [a prebuilt image](https://hub.docker.com/r/hvaghani221/kernel) or modify [Dockerfile](docker/Dockerfile) and create a custom docker image.
```bash
docker run -it -v /home/user/data:/mnt/data -p 8888:8888 hvaghani221/kernel:latest
```
> Note: If you are creating a custom docker image, make sure to specify a static authentication token in the entry point. Refer [Dockerfile](docker/Dockerfile).

### Step 2 - Download binary
Download the binary of your platform from [here](https://github.com/hvaghani221/autopyter/releases).
 
 If you have`golang` installed on your machine, you can install `autopyter` using:
 
```bash
go install github.com/hvaghani221/autopyter@latest
```
Make sure that the binary is executable on your machine.

### Step 3 - Execute binary
Execute the binary as follows:
```bash
./autopyer
```
Here is the list of available configs with default options:
```
Usage of autopyter:
  -address string
        Address to listen on (default "127.0.0.1:8080")
  -debug
        Debug mode enables the logging of kernel activity, directs the logs to the `./logs` directory, and incorporates additional information into the UI.
  -kernelhost string
        Jupyer Server address (default "127.0.0.1:8888")
  -token string
        API token to authenticate with the Jupyter Server(default "ab17a9eb56a95a0bb5af1befa3772368339592c3192da431")
``` 
For example, if you want to serve autopyter frontend on a different port with the debug mode enabled: 
```bash
./autopyter -address "127.0.0.1:5555" -debug
```
