import { Reducer } from 'redux';
import config from './config';
import dependencies from './dependencies';
import embedded from './embedded';
import services from './services';
import trace from './trace';

const reducers: Record<string, Reducer<any, any>> = {
  config,
  dependencies,
  embedded,
  services,
  trace,
};

export default reducers;
