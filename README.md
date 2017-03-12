8ball
=====

Quick and dirty slack bot to add a little humor to a channel.  To install,
clone the repo, then run the following:

    GOPATH=$PWD go get github.com/gorilla/websocket && go build

Copy the binary to its final resting home.  If you would like to add it as a
systemd servce, be sure to modify the executable location and add in your API
token.