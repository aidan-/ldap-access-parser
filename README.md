LDAP Access-log Parser
======================
LDAP Access-log Parser (LAP) is a simple program designed to parse 389 Directory Server (also Fedora Directory Server, Red Hat Directory Server, etc) access logs into individual context aware events ready to be sent to upstream services like ElasticSearch for further analysis.

This application doesn't handle the sending of data to specific endpoints, but that can easily be achieved by piping to something like [log-courier](https://github.com/driskell/log-courier) or [logstash](https://www.elastic.co/products/logstash).

Usage
-----
Use the `-h` flag to view application usage.

```
Usage of ./lap:
  -format string
    	format to output log events.  possible values are 'json' or 'xml'. (default "json")
  -tail
    	tail the log file to receive future events
```

You can begin parsing logs with a simple command:
```
./lap -tail /path/to/access_log
```

This continue to parse the file even if it gets rotated.

Design
------
The applications functionality and output format tries to follow the design specification laid out [here](http://directory.fedoraproject.org/docs/389ds/design/audit-events.html) as much as possible, however there are a few use-cases which were not covered in the design spec that needed to be defined.

More detailed information about the access log format is available in the [Red Hat Directory Server documentation](https://access.redhat.com/documentation/en-US/Red_Hat_Directory_Server/8.1/html/Configuration_and_Command_Reference/logs-reference.html).

Example Output
--------------
Given the raw access log input of:
```
[21/Apr/2009:11:39:55 -0700] conn=14 fd=700 slot=700 connection from 207.1.153.51 to 192.18.122.139
[21/Apr/2009:11:39:55 -0700] conn=14 op=0 BIND dn="" method=sasl version=3 mech=DIGEST-MD5
[21/Apr/2009:11:39:55 -0700] conn=14 op=0 RESULT err=14 tag=97 nentries=0 etime=0, SASL bind in progress
[21/Apr/2009:11:39:55 -0700] conn=14 op=1 BIND dn="uid=jdoe,dc=example,dc=com" method=sasl version=3 mech=DIGEST-MD5
[21/Apr/2009:11:39:55 -0700] conn=14 op=1 RESULT err=0 tag=97 nentries=0 etime=0 dn="uid=jdoe,dc=example,dc=com"
```

The JSON output would look like:
```json
{"time":"21/Apr/2009:11:39:55 -0700","client":"207.1.153.51","server":"192.18.122.139","connection":14,"ssl":false,"operation":0,"authenticateddn":"__anonymous__","action":"BIND","requests":["BIND dn=\"\" method=sasl version=3 mech=DIGEST-MD5"],"responses":["RESULT err=14 tag=97 nentries=0 etime=0, SASL bind in progress"]}
{"time":"21/Apr/2009:11:39:55 -0700","client":"207.1.153.51","server":"192.18.122.139","connection":14,"ssl":false,"operation":1,"authenticateddn":"uid=jdoe,dc=example,dc=com","action":"BIND","requests":["BIND dn=\"uid=jdoe,dc=example,dc=com\" method=sasl version=3 mech=DIGEST-MD5"],"responses":["RESULT err=0 tag=97 nentries=0 etime=0 dn=\"uid=jdoe,dc=example,dc=com\""]}
```

and XML output would look like:
```xml
<Event>
    <DateTime>21/Apr/2009:11:39:55 -0700</DateTime>
    <Client>207.1.153.51</Client>
    <Server>192.18.122.139</Server>
    <Connection>14</Connection>
    <SSL>false</SSL>
    <Operation>0</Operation>
    <AuthenticatedDN>__anonymous__</AuthenticatedDN>
    <Action>BIND</Action>
    <Requests>
        <Request>BIND dn=&#34;&#34; method=sasl version=3 mech=DIGEST-MD5</Request>
    </Requests>
    <Responses>
        <Response>RESULT err=14 tag=97 nentries=0 etime=0, SASL bind in progress</Response>
    </Responses>
</Event>
<Event>
    <DateTime>21/Apr/2009:11:39:55 -0700</DateTime>
    <Client>207.1.153.51</Client>
    <Server>192.18.122.139</Server>
    <Connection>14</Connection>
    <SSL>false</SSL>
    <Operation>1</Operation>
    <AuthenticatedDN>uid=jdoe,dc=example,dc=com</AuthenticatedDN>
    <Action>BIND</Action>
    <Requests>
        <Request>BIND dn=&#34;uid=jdoe,dc=example,dc=com&#34; method=sasl version=3 mech=DIGEST-MD5</Request>
    </Requests>
    <Responses>
        <Response>RESULT err=0 tag=97 nentries=0 etime=0 dn=&#34;uid=jdoe,dc=example,dc=com&#34;</Response>
    </Responses>
</Event>
```

TODO/Notes
-----
- If a connection was initialized before the log monitoring began, the events associated with that connection number will be skipped.  This should probably be a configurable option.
- Connections that disconnect without any operations do not get outputted as an event.  Is this okay?
