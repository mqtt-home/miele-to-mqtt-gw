# Example full message

```json
{
  "ident": {
    "xkmIdentLabel": {
      "releaseVersion": "03.59",
      "techType": "EK037"
    },
    "deviceIdentLabel": {
      "fabNumber": "000101234567",
      "swids": [
        "4551",
        "20492",
        ...
      ],
      "fabIndex": "64",
      "matNumber": "12345678",
      "techType": "G7560"
    },
    "type": {
      "value_raw": 7,
      "key_localized": "Devicetype",
      "value_localized": "Dishwasher"
    },
    "deviceName": ""
  },
  "state": {
    "programType": {
      "value_raw": 0,
      "key_localized": "Programme",
      "value_localized": ""
    },
    "signalInfo": false,
    "dryingStep": {
      "value_raw": null,
      "key_localized": "Drying level",
      "value_localized": ""
    },
    "remainingTime": [
      0,
      2
    ],
    "signalFailure": false,
    "targetTemperature": [
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      },
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      },
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      }
    ],
    "light": 2,
    "ventilationStep": {
      "value_raw": null,
      "key_localized": "Power Level",
      "value_localized": ""
    },
    "remoteEnable": {
      "fullRemoteControl": true,
      "smartGrid": false
    },
    "temperature": [
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      },
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      },
      {
        "unit": "Celsius",
        "value_raw": -32768,
        "value_localized": null
      }
    ],
    "signalDoor": false,
    "startTime": [
      0,
      0
    ],
    "programPhase": {
      "value_raw": 1799,
      "key_localized": "Phase",
      "value_localized": ""
    },
    "status": {
      "value_raw": 5,
      "key_localized": "State",
      "value_localized": "In use"
    },
    "elapsedTime": [
      0,
      0
    ]
  }
}
```