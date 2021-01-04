package de.rnd7.mieletomqtt.miele;

import org.junit.Assert;
import org.junit.Test;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.StringJoiner;

public class FinalJarIntegrationTest {
    private File getFinalJar() {
        return new File("./target/miele-to-mqtt-gw.jar");
    }

    @Test
    public void testFinalJarExists() {
        final File finalJar = getFinalJar();

        Assert.assertTrue(finalJar.isFile());
        Assert.assertTrue(finalJar.exists());
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
        Assert.assertTrue("Expected missing config file", string.contains("ERROR - Expected configuration file as argument"));
        Assert.assertEquals("Exit code", 0, process.waitFor());
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
