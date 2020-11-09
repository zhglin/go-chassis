# Upgrade Guide
## Upgrade from version < 2.0.0 to 2.0.0

### micro service definition
1.Migrate config from "service_description" to "servicecomb.service", for example:

<2.0.0
```yaml
service_description:
  name: Server
  hostname: 10.244.1.3
```
2.0.0
```yaml
servicecomb:
  service:
    name: Server
    hostname: 10.244.1.3
```

2. change "instance_properties" to "instanceProperties", for example:

<2.0.0
```yaml
service_description:
  name: Server
  instance_properties:
    nodeIP: 192.168.0.111
```
2.0.0
```yaml
servicecomb:
  service:
    name: Server
    instanceProperties:
      nodeIP: 192.168.0.111
```

### change cse:// to http://

for example:

<2.0.0
```go
arg, _ := rest.NewRequest("GET", "cse://Server/instances", nil)
```
2.0.0
```go
arg, _ := rest.NewRequest("GET", "http://Server/instances", nil)
```

### change all "cse:"" to "servicecomb:" in yaml
for example:

<2.0.0
```yaml
cse:
 config:
```
2.0.0
```yaml
servicecomb:
 config:
```

### move "registry,router,quota" under "servicecomb"

for example:

<2.0.0
```yaml
cse:
 service:
   registry:
   quota:
   router:
```
2.0.0
```yaml
servicecomb:
 registry:
 quota:
 router:
```

### others

1.if you use archaius.Getxxx("cse.xxxx") to pull config of go chassis

in this case, if you hacked go chassis config to do something, you must change as below

<2.0.0
```go
archaius.Getxxx("cse.xxxx")
```
2.0.0
```go
archaius.Getxxx("servicecomb.xxxx")
```

2.from 1.x to 2.0 there could be many of internal APIs has been refactored, but most API your code won't call. if you find any problem,
please record your problem in [issues](https://github.com/go-chassis/go-chassis/issues).
or even help us to complete this instruction.


## Upgrade from 2.0.0 to 2.0.1
### refactor log tool

you must change import

<=2.0.0
```go
github.com/go-mesh/openlogging
```
>=2.0.1
```go
github.com/go-chassis/openlog
```

please also simplify function name end with f to simple func name and use fmt.Sprintf

<=2.0.0
```go
openlogging.GetLogger().Debugf("init %s's handler map", chainType)
```
>=2.0.1
```go
openlog.Debug(fmt.Sprintf("init %s's handler map", chainType))
```
 
## Upgrade from 2.0.1 to 2.0.2
### rename pkg name
```go
github.com/go-chassis/go-chassis
```
to
```go
github.com/go-chassis/go-chassis/v2
```
Notice: make sure you finish this work by checking go.mod, there should be only github.com/go-chassis/go-chassis/v2 ${version} left

## Upgrade from 2.0.2 to 2.0.3
### log configuration is changed
please refer to https://go-chassis.readthedocs.io/en/v2.0.3/user-guides/log.html