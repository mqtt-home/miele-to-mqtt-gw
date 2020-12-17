package de.rnd7.mieletomqtt.config;

import com.google.gson.JsonDeserializationContext;
import com.google.gson.JsonDeserializer;
import com.google.gson.JsonElement;

import java.lang.reflect.Type;
import java.time.Duration;

public class DurationDeserializer implements JsonDeserializer<Duration> {
    @Override
    public Duration deserialize(JsonElement json, Type typeOfT, JsonDeserializationContext context) {
        return Duration.ofSeconds(json.getAsInt());
    }
}
