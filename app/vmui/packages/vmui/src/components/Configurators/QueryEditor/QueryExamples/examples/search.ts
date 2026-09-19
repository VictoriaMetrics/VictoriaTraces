import { QueryExamplesSection } from "../types";

export const searchExamples: QueryExamplesSection = {
  title: "Search",
  examples: [
    {
      title: "All Spans",
      pattern: "<q>",
      query: "*",
      description: "Show all recorded spans",
      docAnchor: "any-value-filter"
    },
    {
      title: "Filter by Service",
      pattern: "resource_attr:service.name:=\"<service>\"",
      query: "resource_attr:service.name:=\"checkout-service\"",
      description: "Find spans produced by a specific service",
      docAnchor: "exact-filter"
    },
    {
      title: "Filter by Multiple Services",
      pattern: "resource_attr:service.name:in(\"<s1>\", ..., \"<sN>\")",
      query: "resource_attr:service.name:in(\"checkout-service\", \"payment-service\")",
      description: "Find spans produced by any of several services",
      docAnchor: "multi-exact-filter"
    },
    {
      title: "Filter by Operation",
      pattern: "name:=\"<operation>\"",
      query: "name:=\"GET /api/orders\"",
      description: "Find spans for a specific operation or span name",
      docAnchor: "exact-filter"
    },
    {
      title: "Filter by Span Kind",
      pattern: "kind:=\"<SERVER|CLIENT|PRODUCER|CONSUMER|INTERNAL>\"",
      query: "kind:=\"SERVER\"",
      description: "Find spans of a specific OpenTelemetry span kind",
      docAnchor: "exact-filter"
    },
    {
      title: "Find a Trace",
      pattern: "trace_id:=\"<trace_id>\"",
      query: "trace_id:=\"4bf92f3577b34da6a3ce929d0e0e4736\"",
      description: "Get every span belonging to a specific trace",
      docAnchor: "exact-filter"
    },
    {
      title: "Slow Spans",
      pattern: "duration:><nanoseconds>",
      query: "duration:>500000000",
      description: "Find spans slower than a threshold (duration is stored in nanoseconds, so 500ms = 500000000)",
      docAnchor: "range-comparison-filter"
    },
    {
      title: "Errored Spans",
      pattern: "status_code:=\"<code>\"",
      query: "status_code:=\"2\"",
      description: "Find spans that finished with an error status (2 = STATUS_CODE_ERROR per OpenTelemetry)",
      docAnchor: "exact-filter"
    },
  ],
};
