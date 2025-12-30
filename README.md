# ipmipower

Powers on server via IPMI when receives WoL magic packet.

```
$ ./ipmipower -h
Usage of ./ipmipower:
  -host string
        BMC IP address (default is IPMI_HOST environment variable)
  -mac string
        Target MAC address to listen for (format: 00-11-22-33-44-55 or 00:11:22:33:44:55) (default "00:11:22:33:44:55")
  -mode string
        Operation mode: 'wol' for Wake-on-LAN server, 'direct' for direct IPMI command (default "wol")
  -password string
        BMC password (default is IPMI_PASSWORD environment variable)
  -port int
        BMC port (default 623)
  -username string
        BMC username (default is IPMI_USERNAME environment variable)
  -wol-port int
        WoL port to listen on (default 9)
```
