import TNil from './TNil';

type TTraceDiffState = {
  a?: string | TNil;
  b?: string | TNil;
  cohort: string[];
};

export default TTraceDiffState;
