# HL7PROXY
## Proxy for filtering HL7 message data with encoding support
## Description
HL7PROXY is a network proxy focues on HL7 message data.
It transfers HL7 message data from a source port to destination port by using the MLLP.

To prevent corrupt HL7 messages HL7PROXY filters any orphand bytes caused by corrupt HL7 message blocks.

It also supports the charater encoding of incoming or outgoing HL7 message data by dynamically calling an external application.   

HL7PROXY is a command line tool which either can be run in application mode in Windows GUI or as a Windows service in the background.

## Workflow
HL7PROXY must be located between the HL7 communication server and the FORUM server.
 
It has a persistent server socket on which the HL7 communication server can connect and server socket stays connected.

If a data transfer is done from the HL7 communication server via HL7PROXY to the FORUM server HL7PROXY tries to contact the FORUM server. If the connection can be established to the FORUM server the data transfer is done.
 
After a successful data transfer the connection will likely be dropped by the FORUM server. HL7PROXY recognizes this and tries to re-establish the connection to the FORUM server the next time a new data transfer is initiated from the HL7 communication server. 

In case there is a data transfer ongoing and a connection to the FORUM server cannot be stablished the HL7PROXY drops the connection to the HL7 communication server to indicate the unreachable FORUM server. Immediately after the drop the server socket connection is recreated to be available again to the HL7 communication server.

## Installation  
HL7PROXY can be copied to any directory and just executed from there. There is no need for an OS dependent installation.

If HL7PROXY is run as a Windows service please use a local filesystem directory and not a shared location in the network.

## Installation as application
Just copy the package content into any installation directory you would like.
Start the application by calling the executable `hl7proxy.exe` with correct command line parameters.

## Installation as OS service
Follow the instructions "Installation as application".
To register HL7PROXY as an OS service do the following steps.
1. Open a terminal
1. Switch to root/administrative rights
1. CD into your installation directory
1. Installation HL7PROXY as an OS service: 
    1. Windows: `hl7proxy -service install ...`
    1. Linux: `./hl7proxy -service install ...`
1. Uninstallation HL7PROXY as an OS service: 
    1. Windows: `hl7proxy -service uninstall ...`
    1. Linux: `./hl7proxy -service uninstall ...`

## Command line parameters
Parameter | Default value | Description
------------ | ------------- | -------------
? | false | show usage
d |  | destination host address (forumserver:7000)
denc |  | encoder to convert outgoing HL7 messages
f |  | filename to log all transferred HL7 data of stream
hl7 | true | trim data to valid HL7 message blocks in MLLP
log.file |  | filename to log logFile (use "." for D:\go\src\hl7proxy\hl7proxy.log)
log.filesize | 5242880 | max log file size
log.json | false | JSON output
log.verbose | false | verbose logging
s |  | proxy host address:port (:5000)
senc |  | encoder to convert incoming HL7 messages

## Command line parameters
if any encoder application is defined by the "-src-enc" or "-dest-enc" command line parameter the defined encoder application is called in the following notation:

`<encoder> <srcfile> <destfile>`

"srcfile" will be the filename in which the content of the received HL7 data is stored.

The encoder is completely responsible to provide correct encoded data and to store this content to file specified by "destfile".

After the encoder is successfully called by HL7PROXY the file "destfile" will be read by HL7PROXY and the content will be sent.

The encoder must not enhance the content with the MLLP control characters, those will be automatically added by the HL7PROXY. 

## Sample
`hl7proxy -log.verbose -src :6000 -src-enc thai2utf8.bat -dest -dest-enc utf82thai.bat :75000 -file d:\hl7\hl7.log`

### Content of thai2utf8.bat
thaiconv -r %1 -out 3 -w %2

### Content of utf82thai.bat
thaiconv -r %1 -out 1 -w %2
  
## License
All software is copyright and protected by the Apache License, Version 2.0.
https://www.apache.org/licenses/LICENSE-2.0