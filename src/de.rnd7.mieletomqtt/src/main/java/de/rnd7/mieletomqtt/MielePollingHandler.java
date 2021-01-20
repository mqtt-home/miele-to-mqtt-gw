package de.rnd7.mieletomqtt;

import de.rnd7.miele.api.MieleAPI;
import de.rnd7.miele.api.MieleAPIState;
import de.rnd7.miele.api.MieleDevice;
import de.rnd7.miele.api.MieleEventListener;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

public class MielePollingHandler {
    private static final Logger LOGGER = LoggerFactory.getLogger(MielePollingHandler.class);

    private MieleAPI mieleAPI;
    private MieleEventListener listener;

    private final ScheduledExecutorService executor = Executors.newScheduledThreadPool(1);

    public MielePollingHandler(final MieleAPI mieleAPI, final MieleEventListener listener) {
        this.mieleAPI = mieleAPI;
        this.listener = listener;
    }

    public void exec() {
        try {
            for (final MieleDevice device : this.mieleAPI.fetchDevices()) {
                listener.state(MieleAPIState.connected);
                listener.accept(device);
            }
        } catch (final Exception e) {
            listener.state(MieleAPIState.disconnected);
            LOGGER.error(e.getMessage(), e);

            if (!this.mieleAPI.waitReconnect()) {
                executor.shutdown();
            }
        }
    }

    public void start(final long seconds) {
        executor.scheduleAtFixedRate(this::exec, 0, seconds, TimeUnit.SECONDS);

        while (!executor.isTerminated()) {
            try {
                Thread.sleep(1000);
            } catch (InterruptedException e) {
                LOGGER.trace(e.getMessage(), e);
                Thread.currentThread().interrupt();
            }
        }
    }
}
