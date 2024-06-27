# meraki_exporter
A simple prometheus exporter for Meraki appliance.  

env: 
``` 
MERAKI_ORGANIZATION_ID   # meraki organization id. 
MERAKI_API_KEY           # api key, don't forget to whitelist your IP. 
INTERVAL                 # scrape interval. 
```

Docker run: 
``` shell
docker run -p 8080:8080 ghcr.io/alliottech/meraki_exporter:latest -name meraki_exporter
```