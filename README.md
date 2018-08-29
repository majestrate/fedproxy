# fedproxy

socks proxy that reroutes onion and i2p tlds to their own socks proxies while passing through all other tlds 

building:

     $ go get -u github.com/majestrate/fedproxy
     $ cp $(GOPATH)/bin/fedproxy /usr/local/bin/fedproxy
    
usage: 

(just onion)

     $ fedproxy 127.0.0.1:2000 127.0.0.1:9050 
          
(onion + i2pd)

     $ fedproxy 127.0.0.1:2000 127.0.0.1:9050 127.0.0.1:4447
