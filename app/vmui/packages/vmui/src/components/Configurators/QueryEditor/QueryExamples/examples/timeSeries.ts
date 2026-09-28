import { QueryExamplesSection } from "../types";

export const timeSeriesExamples: QueryExamplesSection = {
  title: "Time Series",
  examples: [
    {
      title: "Span Count by Time",
      pattern: "<q> | stats by (_time:<interval>) count() as spans",
      query: "* | stats by (_time:1m) count() as spans",
      description: "Calculate total span count for each time bucket",
      docAnchor: "stats-pipe"
    },
    {
      title: "Error Rate Over Time",
      pattern: `<q> | stats by (_time:<interval>) count() as total,
    count() if (<filter>) as errors
  | math (errors / total) as error_rate`,
      query: `* | stats by (_time:5m) count() as total,
    count() if (status_code:="2") as errors
  | math (errors / total) as error_rate`,
      description: "Track the error rate over time",
      docAnchor: "math-pipe"
    },
    {
      title: "Latency by Time and Service",
      pattern: "<q> | stats by (_time:<interval>, <field>) avg(duration) as avg_duration",
      query: "* | stats by (_time:5m, resource_attr:service.name) avg(duration) as avg_duration",
      description: "Create one latency series per service over time",
      docAnchor: "stats-pipe"
    },
    {
      title: "Throughput for a Service",
      pattern: `<filter> | stats by (_time:<interval>) count() as spans
  | sort by (_time)`,
      query: `resource_attr:service.name:="checkout-service" | stats by (_time:1m) count() as spans
| sort by (_time)`,
      description: "Measure requests per interval for a single service",
      docAnchor: "stats-pipe"
    },
  ],
};
