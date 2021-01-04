package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.json.JSONArray;
import org.junit.Assert;
import org.junit.Test;

public class DurationParserTest extends TestCase {
    @Test
    public void testNull() throws Exception {
        Assert.assertEquals(0,
                DurationParser.parse(null).getSeconds());
    }

    @Test
    public void testInvalid() throws Exception {
        Assert.assertEquals(0,
                DurationParser.parse(new JSONArray("[]")).getSeconds());
        Assert.assertEquals(0,
                DurationParser.parse(new JSONArray("[1, 2, 3, 4]")).getSeconds());
    }

    @Test
    public void testWithSeconds() {
        Assert.assertEquals(3600 + 2 * 60 + 3,
                DurationParser.parse(new JSONArray("[1, 2, 3]")).getSeconds());
    }

    @Test
    public void testWithoutSeconds() {
    	Assert.assertEquals(3600 + 2 * 60,
                DurationParser.parse(new JSONArray("[1, 2]")).getSeconds());
    }
}