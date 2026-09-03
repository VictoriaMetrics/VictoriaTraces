import { QueryExamplesSection } from "../types";

export const formattingExamples: QueryExamplesSection = {
  title: "Formatting",
  examples: [
    {
      title: "Field Selection",
      pattern: "<q> | fields <f1>, ..., <fN>",
      query: "* | fields _time, trace_id, name, duration",
      description: "Display only specific span fields",
      docAnchor: "fields-pipe"
    },
    {
      title: "Sort by Duration",
      pattern: "<q> | sort by (<field>)",
      query: "* | sort by (duration desc)",
      description: "Sort spans by duration, slowest first",
      docAnchor: "sort-pipe"
    },
    {
      title: "Field Rename",
      pattern: "<q> | rename <old> as <new>",
      query: "* | rename resource_attr:service.name as service",
      description: "Give a field a shorter name in the output",
      docAnchor: "rename-pipe"
    },
    {
      title: "Field Deletion",
      pattern: "<q> | delete <f1>, ..., <fN>",
      query: "* | delete span_attr:http.request.header.authorization",
      description: "Remove sensitive or noisy attributes from the output",
      docAnchor: "delete-pipe"
    },
  ],
};
