package de.rnd7.mieletomqtt.config;

import com.google.gson.annotations.SerializedName;

import java.time.Duration;

public class Config {

	private ConfigMqtt mqtt;
	private ConfigMiele miele;

	public ConfigMqtt getMqtt() {
		return mqtt;
	}

	public ConfigMiele getMiele() {
		return miele;
	}

}
