import { QueryExamplesSection } from "../types";

export const aggregationsExamples: QueryExamplesSection = {
  title: "Aggregations",
  examples: [
    {
      title: "Span Count by Service",
      pattern: "<q> | stats by (<field>) count() as spans",
      query: "* | stats by (resource_attr:service.name) count() as spans",
      description: "Count spans grouped by service",
      docAnchor: "stats-pipe"
    },
    {
      title: "Top Services by Span Count",
      pattern: `<q> | stats by (<field>) count() as spans
    | sort by (spans desc)`,
      query: `* | stats by (resource_attr:service.name) count() as spans
  | sort by (spans desc)`,
      description: "Find which services produce the most spans",
      docAnchor: "stats-pipe"
    },
    {
      title: "Average Duration by Operation",
      pattern: "<q> | stats by (<field>) avg(duration) as avg_duration",
      query: "* | stats by (name) avg(duration) as avg_duration",
      description: "Calculate average span duration grouped by operation",
      docAnchor: "stats-pipe-functions"
    },
    {
      title: "p99 Latency by Service",
      pattern: "<q> | stats by (<field>) quantile(<phi>, duration) as p99",
      query: "* | stats by (resource_attr:service.name) quantile(0.99, duration) as p99",
      description: "Calculate the 99th percentile duration per service",
      docAnchor: "stats-pipe-functions"
    },
    {
      title: "Error Count by Service",
      pattern: "<q> | stats by (<field>) count() if (<filter>) as errors",
      query: "* | stats by (resource_attr:service.name) count() if (status_code:=\"2\") as errors",
      description: "Count errored spans grouped by service",
      docAnchor: "stats-pipe"
    },
    {
      title: "Unique Trace Count",
      pattern: "<q> | stats count_uniq(<field>) as traces",
      query: "* | stats count_uniq(trace_id) as traces",
      description: "Count the number of distinct traces matching a filter",
      docAnchor: "stats-pipe-functions"
    },
  ],
};
