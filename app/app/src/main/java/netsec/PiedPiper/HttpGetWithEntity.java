package netsec.PiedPiper;

/**
 * Created by Kyle on 4/25/2017.
 */

import org.apache.http.client.methods.HttpEntityEnclosingRequestBase;
import org.apache.http.client.methods.HttpGet;

import java.net.URI;

public class HttpGetWithEntity extends HttpEntityEnclosingRequestBase {

    public HttpGetWithEntity() {
        super();
    }

    public HttpGetWithEntity(URI uri) {
        super();
        setURI(uri);
    }

    public HttpGetWithEntity(String uri) {
        super();
        setURI(URI.create(uri));
    }

    @Override
    public String getMethod() {
        return HttpGet.METHOD_NAME;
    }
}
