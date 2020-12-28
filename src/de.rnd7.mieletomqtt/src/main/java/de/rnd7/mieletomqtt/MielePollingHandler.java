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

    private String lastMessageJSON = null;
    private MieleAPI mieleAPI;
    private EventBus eventBus;
    private final boolean deduplicate;

    public MielePollingHandler(MieleAPI mieleAPI, EventBus eventBus, boolean deduplicate) {
        this.mieleAPI = mieleAPI;
        this.eventBus = eventBus;
        this.deduplicate = deduplicate;
    }

    public void exec() {
        try {
            for (final MieleDevice device : this.mieleAPI.fetchDevices()) {
                final JSONObject message = device.toFullMessage();
                final String json = message.toString();

                if (sendMessage(json)) {
                    this.lastMessageJSON = json;
                    this.eventBus.post(new Message(device.getId() + "/full", json));
                    this.eventBus.post(new Message(device.getId(), device.toSmallMessage().toString()));
                }
            }
        } catch (final Exception e) {
            LOGGER.error(e.getMessage(), e);
        }
    }

    private boolean sendMessage(final String json) {
        return !deduplicate || this.lastMessageJSON == null || !Objects.equals(this.lastMessageJSON, json);
    }
}
