# Openhab config example

## MQTT.things

```
Bridge mqtt:broker:mosquitto [ host="localhost", port="1883", secure=false, clientID="openhab"] {
	Thing topic Dishwasher "Dishwasher" @ "Kitchen" {
    Channels:
        Type string : state "Dishwasher State" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.state"]
	    Type string : phase "Dishwasher Phase" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.phase"]
    	Type number : phaseId "Dishwasher Phase ID" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.phaseId"]
    	Type string : timeCompleted "Dishwasher ready at" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.timeCompleted"]
    	Type string : remainingDuration "Dishwasher remaining duration" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.remainingDuration"]
    	Type number : remainingDurationMinutes "Dishwasher remaining minutes" [ stateTopic="home/miele/000101234567", transformationPattern="JSONPATH:$.remainingDurationMinutes"]
	}
}
```

## MQTT.items

```
String Dishwasher_State "Status [%s]" { channel="mqtt:topic:mosquitto:Dishwasher:state" }
String Dishwasher_RemainingDuration "Remaining duration [%s h]" { channel="mqtt:topic:mosquitto:Dishwasher:remainingDuration" }
String Dishwasher_TimeComplete "Ready at [%s]" { channel="mqtt:topic:mosquitto:Dishwasher:timeCompleted" }
String Dishwasher_Phase "Phase [%s]" { channel="mqtt:topic:mosquitto:Dishwasher:phase" }
```

## Sitemap

```
Frame label="Dishwasher" visibility=[Dishwasher_State!="OFF"] {
	Default item=Dishwasher_State icon=""
	Default item=Dishwasher_Phase icon=""
	Default item=Dishwasher_RemainingDuration icon="time"
	Default item=Dishwasher_TimeComplete icon="time"
}
```
