import { searchExamples } from "./search";
import { formattingExamples } from "./formatting";
import { aggregationsExamples } from "./aggregations";
import { timeSeriesExamples } from "./timeSeries";
import { QueryExamplesSection } from "../types";

export const queryExamples: QueryExamplesSection[] = [
  searchExamples,
  formattingExamples,
  aggregationsExamples,
  timeSeriesExamples,
];
