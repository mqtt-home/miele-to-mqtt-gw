package de.rnd7.mieletomqtt;

import com.google.common.eventbus.EventBus;

public class Events {
    private final static Events events = new Events();
    private final EventBus eventBus = new EventBus();

    private Events() {

    }

    public static Events getInstance() {
        return events;
    }

    public static void register(final Object object) {
        events.eventBus.register(object);
    }

    public static void post(final Object object) {
        events.eventBus.post(object);
    }

    public static EventBus getBus() {
        return events.eventBus;
    }

    public static void unregister(final Object object) {
        events.eventBus.unregister(object);
    }
}
