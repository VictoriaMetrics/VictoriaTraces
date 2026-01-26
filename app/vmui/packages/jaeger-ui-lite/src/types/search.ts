import { TNil } from '.';

export type SearchQuery = {
  end: number | string;
  limit: number | string;
  lookback: string;
  maxDuration: null | string;
  minDuration: null | string;
  operation: string | TNil;
  service: string;
  start: number | string;
  tags: string | TNil;
};
