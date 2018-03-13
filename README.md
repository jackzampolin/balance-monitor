# `balance-monitor`

This is a configuration driven golang program that fetches balance details for a given BTC address and persists those details, along with the current fee levels in an InfluxDB instance for alerting. 

### Configuration

The [configuration file](/balance-monitor.sample.yaml) contains connection strings for the APIs (`blockchain.info` and `bitcoinfees.earn.com`), an InfluxDB connection and an array of tracked balances:

```yaml
trackedBalances:
  - address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
    service: "satoshi-alert"
    numTransactions: 1000000
```

### Alerting

The `alertThreshold` field in the points written to InfluxDB will be `numTransactions * {{ bitcoinfees.earn.com fastestFee }}`. A kapacitor `tick` script to alert on that number looks like the following:

```js
var db = 'telegraf'

var rp = 'autogen'

var measurement = 'trackedBTCAddresses'

var groupBy = ['address', 'service']

var whereFilter = lambda: TRUE

var name = 'BTC Balance Alerts'

var idVar = name + ':{{.Group}}'

var message = '_*Balance-Monitor*_ 

The following address needs to be refilled:

Address: `{{ index .Tags "address" }}`
Service: `{{ index .Tags "service" }}`
Balance: `{{ index .Fields "balance" }}`
Threshold: `{{ index .Fields "alertThreshold" }}`'

var idTag = 'alertID'

var levelTag = 'level'

var messageField = 'message'

var durationField = 'duration'

var outputDB = 'chronograf'

var outputRP = 'autogen'

var outputMeasurement = 'alerts'

var triggerType = 'threshold'

var data = stream
    |from()
        .database(db)
        .retentionPolicy(rp)
        .measurement(measurement)
        .groupBy(groupBy)
        .where(whereFilter)

var trigger = data
    |alert()
        .crit(lambda: "balance" <= "alertThreshold")
        .stateChangesOnly()
        .message(message)
        .id(idVar)
        .idTag(idTag)
        .levelTag(levelTag)
        .messageField(messageField)
        .durationField(durationField)
        .slack()
        .channel('#my-channel')
        .iconEmoji(':my-emoji:')
        .username('BalanceMonitor')

trigger
    |eval(lambda: float("balance"))
        .as('balance')
        .keep()
    |influxDBOut()
        .create()
        .database(outputDB)
        .retentionPolicy(outputRP)
        .measurement(outputMeasurement)
        .tag('alertName', name)
        .tag('triggerType', triggerType)

trigger
    |httpOut('output')

```