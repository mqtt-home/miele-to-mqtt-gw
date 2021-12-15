package de.rnd7.mieletomqtt.miele;


import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStreamReader;

public class FinalJarIntegrationTest {
    private File getFinalJar() {
        return new File("./target/miele-to-mqtt-gw.jar");
    }

    @Test
    public void testFinalJarExists() {
        final File finalJar = getFinalJar();

        Assertions.assertTrue(finalJar.isFile());
        Assertions.assertTrue(finalJar.exists());
    }

    /**
     * Regressing test for:
     * https://github.com/philipparndt/miele-to-mqtt-gw/issues/25
     *
     * @throws Exception
     */
    @Test
    public void execFinalJar() throws Exception {
        final Process process = startFinalJar();

        final String string = consumeInputStream(process);
        Assertions.assertTrue(string.contains("ERROR - Expected configuration file as argument"), "Expected missing config file");
        Assertions.assertEquals(0, process.waitFor(), "Exit code");
    }

    private Process startFinalJar() throws IOException {
        final File finalJar = getFinalJar();
        final String java = System.getProperty("java.home") + "/bin/java";

        return new ProcessBuilder(java,
            "-jar", finalJar.getAbsolutePath()
        )
            .start();
    }

    private String consumeInputStream(final Process process) throws IOException {
        final StringBuilder sb = new StringBuilder();
        try (final BufferedReader in = new BufferedReader(new InputStreamReader(process.getInputStream()))) {
            String s = "";
            while ((s = in.readLine()) != null) {
                sb.append(s);
                sb.append(System.lineSeparator());
            }
        }

        return sb.toString();
    }

}
