#! /usr/bin/bash
go build
sudo service goOpenhab stop
cp goOpenhab /opt/home*
sudo service goOpenhab start
tail -f /opt/home*/log/go*.log

