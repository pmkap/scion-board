# SCiON Board
Simple Chat and Game Board Server for the [SCIONLab](https://www.scionlab.org/) network. I made this to learn about Go and its concurrency model and playing around with SCIONLab at the same time.

## Features
* Chat
* Wordle (Credits to [Ashish Shenoy](https://github.com/AshishShenoy/wordle), MIT License)

## Connect
You just need a machine connected to the SCIONLab network with [scion-netcat](https://github.com/netsec-ethz/scion-apps) installed. There is a server running (or not) on 17-ffaa:1:f7f,127.0.0.1 port 1337. Simply connect with:

`scion-netcat 17-ffaa:1:f7f,127.0.0.1:1337`

## Build and Run
`go run board` to run on port 1337.
