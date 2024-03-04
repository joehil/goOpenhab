package main

import (
	"github.com/patrickmn/go-cache"
)

type Msginfo struct {
	Msgdate     string
	Msgtime     string
	Msgevent    string
	Msgobjtype  string
	Msgobject   string
	Msgoldstate string
	Msgnewstate string
}

type Msgwarn struct {
	Msgdate  string
	Msgtime  string
	Msgevent string
	Msgtext  string
}

type Mqttparms struct {
	Topic   string
	Message string
}

type Requestin struct {
	Node  string
	Item  string
	Value string
	Data  string
}

type Generalvars struct {
	Pers       *cache.Cache
	Telegram   chan string
	Tbtoken    string
	Chatid     int64
	Mqttmsg    chan Mqttparms
	Mqttbroker string
	Resturl    string
	Resttoken  string
	Getin      chan Requestin
	Getout     chan string
	Postin     chan Requestin
}
