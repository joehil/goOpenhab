#!/bin/bash
# Script Name:  fritzbox-restart.v1.sh
# Beschreibung: startet die Fritzbox neu
#               Dieses Bash-Script nutzt das Protokoll TR-064
#               Skript funktioniert für alle Fritzboxen ab FritzOS 6.0
# Aufruf:       ping -c 1 1.1.1.1 >/dev/null || (for i in {1..3}; do ping -c 1 1.1.1.1 >/dev/null && exit; sleep 30; done; /home/scripts/network/fritzbox-reboot.v1.sh)
# Aufruf 2      bash ./fritzbox-reboot.v1.sh
# Autor:        Patrick Asmus
# Web:          https://www.media-techport.de
# Git-Reposit.: https://git.media-techport.de/scriptos/fritzbox-restart-script
# Version:      1.0.2
# Datum:        16.07.2023
# Modifikation: Logging hinzugefuegt
#####################################################
# Variablen
IPS="<ip of fritz router>"
FRITZ_USER="<user>"
FRITZ_PW="<password>"
LOG_FILE="<path to log>"
# Funktion zum Schreiben von Logs
log() {
    timestamp=$(date +"%Y-%m-%d %T")
    echo "[${timestamp}] $1" >> "$LOG_FILE"
}
# Ausführung
location="/upnp/control/deviceconfig"
uri="urn:dslforum-org:service:DeviceConfig:1"
action='Reboot'
log "Script gestartet."
for IP in ${IPS}; do
    log "Starte Neustart für Fritzbox mit IP: $IP"
    curl -k -m 5 --anyauth -u "$FRITZ_USER:$FRITZ_PW" "http://$IP:49000$location" -H 'Content-Type: text/xml; charset="utf-8"' -H "SoapAction:$uri#$action" -d "<?xml version='1.0' encoding='utf-8'?><s:Envelope s:encodingStyle='http://schemas.xmlsoap.org/soap/encoding/' xmlns:s='http://schemas.xmlsoap.org/soap/envelope/'><s:Body><u:$action xmlns:u='$uri'></u:$action></s:Body></s:Envelope>" -s > /dev/null
    log "Neustart für Fritzbox mit IP: $IP abgeschlossen."
done
log "Script beendet."
