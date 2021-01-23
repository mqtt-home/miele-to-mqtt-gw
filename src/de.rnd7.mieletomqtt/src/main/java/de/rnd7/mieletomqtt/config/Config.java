package de.rnd7.mieletomqtt.config;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.mqttgateway.config.ConfigMqtt;

public class Config {

    private ConfigMqtt mqtt = new ConfigMqtt();
    private ConfigMiele miele = new ConfigMiele();

    public ConfigMqtt getMqtt() {
        return mqtt;
    }

    public ConfigMiele getMiele() {
        return miele;
    }
}
