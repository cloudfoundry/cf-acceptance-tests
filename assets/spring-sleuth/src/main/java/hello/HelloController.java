package hello;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;
import jakarta.servlet.http.HttpServletRequest;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class HelloController {
    private static final Logger LOGGER = LoggerFactory.getLogger(HelloController.class);

    private final Tracer tracer;

    public HelloController(Tracer tracer) {
        this.tracer = tracer;
    }

    @RequestMapping("/")
    public String index(HttpServletRequest request) {
        Span currentSpan = tracer.currentSpan();
        if (currentSpan == null) {
            return "current span: [Trace: none, Span: none, exportable:unknown] \n parents: []";
        }

        List<String> parentsHex = parentSpanIds(request);
        String traceId = currentSpan.context().traceId();
        String spanId = currentSpan.context().spanId();
        String exportable = String.valueOf(currentSpan.context().sampled());

        LOGGER.info("handling request");
        return "current span: [Trace: " + traceId + ", Span: " + spanId + ", exportable:" + exportable + "] \n parents: " + parentsHex;
    }

    private List<String> parentSpanIds(HttpServletRequest request) {
        Set<String> parentIds = new LinkedHashSet<>();

        String xB3ParentSpanId = request.getHeader("X-B3-ParentSpanId");
        if (isPresent(xB3ParentSpanId)) {
            parentIds.add(xB3ParentSpanId.toLowerCase());
        }

        String b3Single = request.getHeader("b3");
        if (isPresent(b3Single)) {
            String[] parts = b3Single.split("-");
            if (parts.length >= 4 && isPresent(parts[3])) {
                parentIds.add(parts[3].toLowerCase());
            }
        }

        String traceparent = request.getHeader("traceparent");
        if (isPresent(traceparent)) {
            String[] parts = traceparent.split("-");
            if (parts.length == 4 && isPresent(parts[2])) {
                parentIds.add(parts[2].toLowerCase());
            }
        }

        return new ArrayList<>(parentIds);
    }

    private boolean isPresent(String value) {
        return value != null && !value.isBlank();
    }
}

