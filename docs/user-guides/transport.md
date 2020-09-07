# Transport

## Introduction
this part define the settings related about network communication

## Configurations

**transport.failure.{protocol_name}**
> *(required, string)* the name of the protocol client, now only support rest failure. a string list connect with comma,each string 
starts with http_, combine with http status code.
you can define what can be considered as failure 
and make it count in circuit breaker and fault-tolerance module


**transport.maxIdleCon.{protocol_name}**
> *(required, string)* MaxIdleConns controls the maximum number of idle (keep-alive) connections 
across all hosts. Zero means no limit. it only works for rest protocol


**transport.maxBodyBytes.{protocol_name}**
> *(optional, int, (bytes))* maxBodyBytes controls the maximum number of request body size , 
 Zero means no limit. it only works for rest protocol.

**transport.maxHeaderBytes.{protocol_name}**
> *(optional, int, (bytes))* maxHeaderBytes controls the maximum number of request header keys/values, including the 
request line. Zero means no limit. It only works for rest protocol.

**transport.timeout.{protocol_name}**
> *(optional, string)* timeout controls the timeout of the server. Use Golang duration string.

## Example
The cases of http_500,http_502 are considered as unsuccessful attempts
```
servicecomb:
  transport:
    failure:
      rest: http_500,http_502
    maxIdleCon:
      rest: 1024
    maxBodyBytes:
      rest: 1
    maxHeaderBytes:
      rest: 1
    timeout:
      rest: 30s
```
