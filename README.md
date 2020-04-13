# fedproxy

routes `.onion` and clearnet domains over tor, `.i2p` domains over i2p, and `.loki` domains over lokinet if it's configured.

building:

     $ go get -u github.com/majestrate/fedproxy
     $ cp $(GOPATH)/bin/fedproxy /usr/local/bin/fedproxy
    
usage: 

     $ fedproxy socks 127.0.0.1:2000 127.0.0.1:9050 127.0.0.1:4447

then use socks proxy at `127.0.0.1:2000`
