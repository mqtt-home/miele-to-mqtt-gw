package de.rnd7.mieletomqtt.config;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.mqtt.ConfigMqtt;

public class Config {

    private ConfigMqtt mqtt = new ConfigMqtt();
    private ConfigMiele miele = new ConfigMiele();
    private boolean deduplicate = false;

    public ConfigMqtt getMqtt() {
        return mqtt;
    }

    public ConfigMiele getMiele() {
        return miele;
    }

    public boolean isDeduplicate() {
        return deduplicate;
    }
}
