package de.rnd7.mieletomqtt;

import de.rnd7.miele.api.MieleAPI;
import de.rnd7.miele.api.MieleDevice;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

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
