package de.rnd7.mieletomqtt;

import com.google.common.eventbus.EventBus;
import de.rnd7.miele.api.MieleAPI;
import de.rnd7.miele.api.MieleDevice;
import de.rnd7.mqtt.Message;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Objects;

public class MielePollingHandler {
    private static final Logger LOGGER = LoggerFactory.getLogger(MielePollingHandler.class);

    private MieleAPI mieleAPI;
    private MieleEventHandler handler;

    public MielePollingHandler(MieleAPI mieleAPI, MieleEventHandler handler) {
        this.mieleAPI = mieleAPI;
        this.handler = handler;
    }

    public void exec() {
        try {
            for (final MieleDevice device : this.mieleAPI.fetchDevices()) {
                handler.accept(device);
            }
        } catch (final Exception e) {
            LOGGER.error(e.getMessage(), e);
        }
    }
}
